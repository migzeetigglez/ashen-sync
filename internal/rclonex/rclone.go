package rclonex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Client shells out to rclone (bundled next to Save Channeler, or system-installed).
// Save Channeler never embeds cloud OAuth secrets.
type Client struct {
	Bin string
}

var (
	resolvedBin     string
	resolvedBinOnce sync.Once
)

func (c Client) bin() string {
	name := c.Bin
	if name == "" {
		name = "rclone"
	}
	// Absolute / relative path in config: use as-is when it exists.
	if filepath.IsAbs(name) || strings.Contains(name, string(os.PathSeparator)) {
		if st, err := os.Stat(name); err == nil && !st.IsDir() {
			return name
		}
		// Fall through to resolution if configured path is missing.
	}
	resolvedBinOnce.Do(func() {
		resolvedBin = resolveRclone(rcloneBaseName(name))
	})
	if resolvedBin != "" {
		return resolvedBin
	}
	return name
}

// ResolvedPath returns the rclone binary path that will be executed.
func (c Client) ResolvedPath() string {
	return c.bin()
}

func rcloneBaseName(name string) string {
	base := filepath.Base(name)
	if base == "" || base == "." || base == string(os.PathSeparator) {
		return "rclone"
	}
	return base
}

func resolveRclone(name string) string {
	exeName := name
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(exeName), ".exe") {
		exeName = exeName + ".exe"
	}

	// 1) Bundled next to Save Channeler (preferred — no PATH required).
	if exeDir := executableDir(); exeDir != "" {
		for _, p := range []string{
			filepath.Join(exeDir, exeName),
			filepath.Join(exeDir, name),
			filepath.Join(exeDir, "bin", exeName),
			filepath.Join(exeDir, "bin", name),
		} {
			if isExecFile(p) {
				return p
			}
		}
	}

	// 2) PATH
	for _, cand := range []string{exeName, name} {
		if p, err := exec.LookPath(cand); err == nil {
			return p
		}
	}

	// 3) Common user / system locations
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", exeName),
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, "bin", exeName),
			filepath.Join(home, "bin", name),
		)
	}
	candidates = append(candidates,
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
	)
	for _, p := range candidates {
		if isExecFile(p) {
			return p
		}
	}
	return name
}

func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

func isExecFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func (c Client) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.bin(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("rclone %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func (c Client) Version(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "version")
	if err != nil {
		return "", err
	}
	line := strings.Split(out, "\n")[0]
	return strings.TrimSpace(line), nil
}

func (c Client) ListRemotes(ctx context.Context) ([]string, error) {
	out, err := c.run(ctx, "listremotes")
	if err != nil {
		return nil, err
	}
	var remotes []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ":")
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

func (c Client) About(ctx context.Context, remoteRoot string) error {
	_, err := c.run(ctx, "about", remoteRoot)
	return err
}

// CopyToRemote uploads localDir contents into remoteDir (rclone remote:path).
func (c Client) CopyToRemote(ctx context.Context, localDir, remoteDir string) error {
	_, err := c.run(ctx, "copy", localDir, remoteDir, "--create-empty-src-dirs", "-v")
	return err
}

// CopyFromRemote downloads remoteDir into localDir.
func (c Client) CopyFromRemote(ctx context.Context, remoteDir, localDir string) error {
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return err
	}
	_, err := c.run(ctx, "copy", remoteDir, localDir, "--create-empty-src-dirs", "-v")
	return err
}

// Lsf lists files in a remote directory (names only).
func (c Client) Lsf(ctx context.Context, remoteDir string) ([]string, error) {
	out, err := c.run(ctx, "lsf", remoteDir)
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "directory not found") || strings.Contains(low, "not found") {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}
