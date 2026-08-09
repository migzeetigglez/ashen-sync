//go:build !windows

package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

func applyPlatform(ctx context.Context, info Info, stage Stage) error {
	_ = ctx
	if runtime.GOOS != "linux" {
		stage.Cleanup()
		return fmt.Errorf("in-app updates are not supported on %s", runtime.GOOS)
	}

	exePath, err := ExecutablePath()
	if err != nil {
		stage.Cleanup()
		return err
	}
	installDir, err := InstallDir()
	if err != nil {
		stage.Cleanup()
		return err
	}
	notesPath, err := WriteNotesFile(info)
	if err != nil {
		stage.Cleanup()
		return err
	}

	newBin := exePath + ".new"
	if err := copyFile(stage.BinaryPath, newBin); err != nil {
		stage.Cleanup()
		return err
	}
	if err := os.Chmod(newBin, 0o755); err != nil {
		stage.Cleanup()
		return err
	}

	if err := os.Rename(newBin, exePath); err != nil {
		// Fallback: shell helper after exit.
		if err := writeUnixHelper(unixHelperArgs{
			PID:        os.Getpid(),
			ExePath:    exePath,
			NewBin:     newBin,
			RcloneSrc:  stage.RclonePath,
			RcloneDest: filepath.Join(installDir, "rclone"),
			NotesPath:  notesPath,
			StageDir:   stage.Dir,
		}); err != nil {
			stage.Cleanup()
			return err
		}
		return nil
	}

	if stage.RclonePath != "" {
		rcloneDest := filepath.Join(installDir, "rclone")
		tmp := rcloneDest + ".new"
		if err := copyFile(stage.RclonePath, tmp); err == nil {
			_ = os.Chmod(tmp, 0o755)
			_ = os.Rename(tmp, rcloneDest)
		}
	}

	stage.Cleanup()

	args := []string{exePath, "gui", "--whats-new", notesPath}
	return syscall.Exec(exePath, args, os.Environ())
}

type unixHelperArgs struct {
	PID        int
	ExePath    string
	NewBin     string
	RcloneSrc  string
	RcloneDest string
	NotesPath  string
	StageDir   string
}

func writeUnixHelper(a unixHelperArgs) error {
	helper := a.ExePath + ".update.sh"
	script := fmt.Sprintf(`#!/bin/sh
set -eu
PID=%d
EXE=%q
NEW=%q
NOTES=%q
STAGEDIR=%q
RCLONE_SRC=%q
RCLONE_DEST=%q
while kill -0 "$PID" 2>/dev/null; do sleep 1; done
if [ -f "$NEW" ]; then
  mv -f "$NEW" "$EXE"
  chmod +x "$EXE"
fi
if [ -n "$RCLONE_SRC" ] && [ -f "$RCLONE_SRC" ]; then
  cp -f "$RCLONE_SRC" "$RCLONE_DEST"
  chmod +x "$RCLONE_DEST"
fi
rm -rf "$STAGEDIR"
"$EXE" gui --whats-new "$NOTES" &
rm -f %q
`, a.PID, a.ExePath, a.NewBin, a.NotesPath, a.StageDir, a.RcloneSrc, a.RcloneDest, helper)

	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", helper)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
