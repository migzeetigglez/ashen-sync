package rclonex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client shells out to a user-installed rclone binary.
// Save Channeler never embeds cloud OAuth secrets.
type Client struct {
	Bin string
}

func (c Client) bin() string {
	if c.Bin == "" {
		return "rclone"
	}
	return c.Bin
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
