package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Stage holds extracted update files ready to install.
type Stage struct {
	Dir        string // temp staging directory
	BinaryPath string // path to new save-channeler binary
	RclonePath string // optional path to new rclone
	GOOS       string
}

// Cleanup removes the staging directory.
func (s Stage) Cleanup() {
	if s.Dir != "" {
		_ = os.RemoveAll(s.Dir)
	}
}

// DownloadAndStage fetches the release asset and extracts the app (+ rclone) binaries.
func DownloadAndStage(ctx context.Context, info Info) (Stage, error) {
	return downloadAndStage(ctx, info, runtime.GOOS, nil)
}

func downloadAndStage(ctx context.Context, info Info, goos string, client *http.Client) (Stage, error) {
	if info.AssetURL == "" {
		return Stage{}, fmt.Errorf("no download URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return Stage{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		return Stage{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Stage{}, fmt.Errorf("download: HTTP %d", res.StatusCode)
	}

	tmp, err := os.MkdirTemp("", "save-channeler-update-*")
	if err != nil {
		return Stage{}, err
	}
	stage := Stage{Dir: tmp, GOOS: goos}

	archivePath := filepath.Join(tmp, info.AssetName)
	f, err := os.Create(archivePath)
	if err != nil {
		stage.Cleanup()
		return Stage{}, err
	}
	if _, err := io.Copy(f, res.Body); err != nil {
		f.Close()
		stage.Cleanup()
		return Stage{}, err
	}
	if err := f.Close(); err != nil {
		stage.Cleanup()
		return Stage{}, err
	}

	extractDir := filepath.Join(tmp, "extracted")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		stage.Cleanup()
		return Stage{}, err
	}

	switch {
	case strings.HasSuffix(info.AssetName, ".zip"):
		if err := unzip(archivePath, extractDir); err != nil {
			stage.Cleanup()
			return Stage{}, err
		}
	case strings.HasSuffix(info.AssetName, ".tar.gz") || strings.HasSuffix(info.AssetName, ".tgz"):
		if err := untarGz(archivePath, extractDir); err != nil {
			stage.Cleanup()
			return Stage{}, err
		}
	default:
		stage.Cleanup()
		return Stage{}, fmt.Errorf("unsupported archive: %s", info.AssetName)
	}

	binName := BinaryName(goos)
	rcloneName := RcloneName(goos)
	binPath, err := findFile(extractDir, binName)
	if err != nil {
		stage.Cleanup()
		return Stage{}, fmt.Errorf("archive missing %s: %w", binName, err)
	}
	stage.BinaryPath = binPath
	if rp, err := findFile(extractDir, rcloneName); err == nil {
		stage.RclonePath = rp
	}
	return stage, nil
}

func findFile(root, name string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		if err != nil && err != filepath.SkipAll {
			return "", err
		}
		return "", fmt.Errorf("not found")
	}
	return found, nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if err := extractZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, dest string) error {
	name := filepath.Clean(f.Name)
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid zip path: %s", f.Name)
	}
	target := filepath.Join(dest, name)
	if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) && target != filepath.Clean(dest) {
		return fmt.Errorf("zip path escapes dest: %s", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func untarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("invalid tar path: %s", hdr.Name)
		}
		target := filepath.Join(dest, name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) && target != filepath.Clean(dest) {
			return fmt.Errorf("tar path escapes dest: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
