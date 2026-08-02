package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
)

const AppName = "save-channeler"
const legacyAppName = "ashen-sync"

type Config struct {
	RclonePath         string            `toml:"rclone_path"`
	Remote             string            `toml:"remote"`              // e.g. "gdrive"
	RemotePath         string            `toml:"remote_path"`         // e.g. "save-channeler"
	SnapshotKeep       int               `toml:"snapshot_keep"`       // default 10
	QuietPeriodSeconds int               `toml:"quiet_period_seconds"` // watcher debounce
	PullIntervalSeconds int              `toml:"pull_interval_seconds"` // idle remote check
	GameOverrides      map[string]string `toml:"game_overrides"`      // game id -> save dir
	SteamExecutable    string            `toml:"steam_executable"`
}

func Default() Config {
	return Config{
		RclonePath:          "rclone",
		RemotePath:          "save-channeler",
		SnapshotKeep:        10,
		QuietPeriodSeconds:  15,
		PullIntervalSeconds: 300,
		GameOverrides:       map[string]string{},
	}
}

func (c Config) QuietPeriod() time.Duration {
	if c.QuietPeriodSeconds <= 0 {
		return 15 * time.Second
	}
	return time.Duration(c.QuietPeriodSeconds) * time.Second
}

func (c Config) PullInterval() time.Duration {
	if c.PullIntervalSeconds <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.PullIntervalSeconds) * time.Second
}

func Dir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			return "", errors.New("APPDATA is not set")
		}
		return filepath.Join(base, AppName), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", AppName), nil
}

func DataDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("APPDATA")
		}
		if base == "" {
			return "", errors.New("LOCALAPPDATA/APPDATA is not set")
		}
		return filepath.Join(base, AppName), nil
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", AppName), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to pre-rename config location.
			if legacy, lerr := legacyPath(); lerr == nil {
				if data, err = os.ReadFile(legacy); err == nil {
					path = legacy
				} else {
					return cfg, os.ErrNotExist
				}
			} else {
				return cfg, os.ErrNotExist
			}
		} else {
			return cfg, err
		}
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.SnapshotKeep <= 0 {
		cfg.SnapshotKeep = 10
	}
	if cfg.QuietPeriodSeconds <= 0 {
		cfg.QuietPeriodSeconds = 15
	}
	if cfg.PullIntervalSeconds <= 0 {
		cfg.PullIntervalSeconds = 300
	}
	if cfg.RclonePath == "" {
		cfg.RclonePath = "rclone"
	}
	if cfg.RemotePath == "" {
		cfg.RemotePath = "save-channeler"
	}
	if cfg.GameOverrides == nil {
		cfg.GameOverrides = map[string]string{}
	}
	_ = path
	return cfg, nil
}

func legacyPath() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			return "", errors.New("APPDATA is not set")
		}
		return filepath.Join(base, legacyAppName, "config.toml"), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, legacyAppName, "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", legacyAppName, "config.toml"), nil
}

func (c Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(c)
}

func (c Config) RemoteRoot() string {
	if c.Remote == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s", c.Remote, c.RemotePath)
}

func (c Config) Validate() error {
	if c.Remote == "" {
		return errors.New("remote is not set; run `save-channeler init`")
	}
	return nil
}

// ClearRemote unsets the sync destination and saves config.
// rclone remotes themselves are left alone so other tools keep working.
func (c *Config) ClearRemote() error {
	c.Remote = ""
	c.RemotePath = "save-channeler"
	return c.Save()
}
