package snapshot

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Store struct {
	Root string
	Keep int
}

func (s Store) Capture(gameID, srcDir string) (string, error) {
	if s.Keep <= 0 {
		s.Keep = 10
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	dest := filepath.Join(s.Root, "snapshots", gameID, stamp)
	if err := copyDir(srcDir, dest); err != nil {
		return "", err
	}
	_ = s.prune(gameID)
	return dest, nil
}

func (s Store) prune(gameID string) error {
	dir := filepath.Join(s.Root, "snapshots", gameID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	if len(dirs) <= s.Keep {
		return nil
	}
	for _, name := range dirs[:len(dirs)-s.Keep] {
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func Restore(srcSnapshot, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("clear dest: %w", err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return copyDir(srcSnapshot, destDir)
}
