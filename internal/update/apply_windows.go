//go:build windows

package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func applyPlatform(ctx context.Context, info Info, stage Stage) error {
	_ = ctx
	installDir, err := InstallDir()
	if err != nil {
		stage.Cleanup()
		return err
	}
	exePath, err := ExecutablePath()
	if err != nil {
		stage.Cleanup()
		return err
	}
	notesPath, err := WriteNotesFile(info)
	if err != nil {
		stage.Cleanup()
		return err
	}

	stagingBin := filepath.Join(installDir, "save-channeler.update.exe")
	if err := copyFile(stage.BinaryPath, stagingBin); err != nil {
		stage.Cleanup()
		return err
	}
	var stagingRclone string
	if stage.RclonePath != "" {
		stagingRclone = filepath.Join(installDir, "rclone.update.exe")
		if err := copyFile(stage.RclonePath, stagingRclone); err != nil {
			stage.Cleanup()
			return err
		}
	}

	batPath := filepath.Join(installDir, "update-save-channeler.bat")
	bat := buildWindowsUpdateBat(windowsUpdateBatArgs{
		PID:           os.Getpid(),
		ExePath:       exePath,
		StagingBin:    stagingBin,
		StagingRclone: stagingRclone,
		RcloneDest:    filepath.Join(installDir, "rclone.exe"),
		NotesPath:     notesPath,
		StageDir:      stage.Dir,
		BatPath:       batPath,
	})
	if err := os.WriteFile(batPath, []byte(bat), 0o755); err != nil {
		stage.Cleanup()
		return err
	}

	cmd := exec.Command("cmd", "/C", "start", "", batPath)
	cmd.Dir = installDir
	if err := cmd.Start(); err != nil {
		stage.Cleanup()
		return fmt.Errorf("start update helper: %w", err)
	}
	return nil
}

type windowsUpdateBatArgs struct {
	PID           int
	ExePath       string
	StagingBin    string
	StagingRclone string
	RcloneDest    string
	NotesPath     string
	StageDir      string
	BatPath       string
}

func buildWindowsUpdateBat(a windowsUpdateBatArgs) string {
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("setlocal\r\n")
	fmt.Fprintf(&b, "set \"PID=%d\"\r\n", a.PID)
	fmt.Fprintf(&b, "set \"EXE=%s\"\r\n", a.ExePath)
	fmt.Fprintf(&b, "set \"NEW=%s\"\r\n", a.StagingBin)
	fmt.Fprintf(&b, "set \"NOTES=%s\"\r\n", a.NotesPath)
	fmt.Fprintf(&b, "set \"STAGEDIR=%s\"\r\n", a.StageDir)
	fmt.Fprintf(&b, "set \"BAT=%s\"\r\n", a.BatPath)
	b.WriteString(":wait\r\n")
	b.WriteString("tasklist /FI \"PID eq %PID%\" 2>NUL | find \"%PID%\" >NUL\r\n")
	b.WriteString("if not errorlevel 1 (\r\n")
	b.WriteString("  timeout /t 1 /nobreak >NUL\r\n")
	b.WriteString("  goto wait\r\n")
	b.WriteString(")\r\n")
	b.WriteString("copy /Y \"%NEW%\" \"%EXE%\" >NUL\r\n")
	b.WriteString("del /F /Q \"%NEW%\" >NUL 2>NUL\r\n")
	if a.StagingRclone != "" {
		fmt.Fprintf(&b, "set \"NEWRCLONE=%s\"\r\n", a.StagingRclone)
		fmt.Fprintf(&b, "set \"RCLONE=%s\"\r\n", a.RcloneDest)
		b.WriteString("if exist \"%NEWRCLONE%\" (\r\n")
		b.WriteString("  copy /Y \"%NEWRCLONE%\" \"%RCLONE%\" >NUL\r\n")
		b.WriteString("  del /F /Q \"%NEWRCLONE%\" >NUL 2>NUL\r\n")
		b.WriteString(")\r\n")
	}
	b.WriteString("start \"\" \"%EXE%\" gui --whats-new \"%NOTES%\"\r\n")
	b.WriteString("rmdir /S /Q \"%STAGEDIR%\" >NUL 2>NUL\r\n")
	b.WriteString("del /F /Q \"%BAT%\" >NUL 2>NUL\r\n")
	return b.String()
}
