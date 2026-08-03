package syncer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/migzeetigglez/ashen-sync/internal/config"
	"github.com/migzeetigglez/ashen-sync/internal/detect"
	"github.com/migzeetigglez/ashen-sync/internal/manifest"
	"github.com/migzeetigglez/ashen-sync/internal/rclonex"
	"github.com/migzeetigglez/ashen-sync/internal/snapshot"
)

const ToolVersion = "0.1.0"

type Prefer string

const (
	PreferAbort  Prefer = ""
	PreferLocal  Prefer = "local"
	PreferRemote Prefer = "remote"
)

type ConflictError struct {
	GameID       string
	LocalHash    string
	RemoteHash   string
	LastSyncHash string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict for %s: local=%s remote=%s last_sync=%s",
		e.GameID, short(e.LocalHash), short(e.RemoteHash), short(e.LastSyncHash))
}

func short(h string) string {
	if len(h) < 12 {
		return h
	}
	return h[:12]
}

type Engine struct {
	Cfg    config.Config
	Rclone rclonex.Client
	Snap   snapshot.Store
	Log    func(string)
}

func New(cfg config.Config) (Engine, error) {
	data, err := config.DataDir()
	if err != nil {
		return Engine{}, err
	}
	return Engine{
		Cfg:    cfg,
		Rclone: rclonex.Client{Bin: cfg.RclonePath},
		Snap:   snapshot.Store{Root: data, Keep: cfg.SnapshotKeep},
		Log:    func(string) {},
	}, nil
}

func (e Engine) logf(format string, args ...any) {
	if e.Log != nil {
		e.Log(fmt.Sprintf(format, args...))
	}
}

func (e Engine) remoteGameDir(loc detect.SaveLocation) string {
	root := e.Cfg.RemoteRoot()
	return fmt.Sprintf("%s/%s/%s", root, loc.Game.RemoteKey, loc.SteamID)
}

func (e Engine) statePath(gameID string) (string, error) {
	data, err := config.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(data, "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, gameID+".json"), nil
}

func (e Engine) loadLastSync(gameID string) (manifest.Manifest, error) {
	path, err := e.statePath(gameID)
	if err != nil {
		return manifest.Manifest{}, err
	}
	m, err := manifest.Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest.Manifest{}, nil
		}
		return manifest.Manifest{}, err
	}
	return m, nil
}

func (e Engine) saveLastSync(m manifest.Manifest) error {
	path, err := e.statePath(m.GameID)
	if err != nil {
		return err
	}
	return m.Write(path)
}

type Status struct {
	Location   detect.SaveLocation
	LocalHash  string
	RemoteHash string
	LastSync   string
	InSync     bool
	HasRemote  bool
}

func (e Engine) Status(ctx context.Context, gameID string) (Status, error) {
	loc, err := detect.FindGame(gameID, e.Cfg.GameOverrides)
	if err != nil {
		return Status{}, err
	}
	st := Status{Location: loc}
	if !loc.Exists {
		return st, nil
	}
	local, err := manifest.Build(loc.Game.ID, loc.Game.RemoteKey, loc.SteamID, ToolVersion, loc.Path)
	if err != nil {
		return st, err
	}
	st.LocalHash = local.Aggregate

	last, _ := e.loadLastSync(loc.Game.ID)
	st.LastSync = last.SyncHash()

	tmp, err := os.MkdirTemp("", "ashen-remote-*")
	if err != nil {
		return st, err
	}
	defer os.RemoveAll(tmp)

	remoteDir := e.remoteGameDir(loc)
	// Probe with lsf first — missing remote dirs are normal on first push and
	// must not dump rclone "directory not found" ERROR spam into the activity log.
	listing, err := e.Rclone.Lsf(ctx, remoteDir)
	if err != nil {
		e.logf("remote check failed: %v", err)
		return st, nil
	}
	if len(listing) == 0 {
		e.logf("no cloud save yet for %s (first push will create it)", loc.Game.ID)
		return st, nil
	}
	if err := e.Rclone.CopyFromRemote(ctx, remoteDir, tmp); err != nil {
		e.logf("remote download for status failed: %v", err)
		return st, nil
	}
	names, _ := os.ReadDir(tmp)
	if len(names) == 0 {
		return st, nil
	}
	st.HasRemote = true
	// Always hash remote *content*. Trusting remote manifest.json.Aggregate caused
	// false conflicts when the manifest went stale relative to DS30000.sl2.
	agg, err := manifest.ContentAggregate(tmp)
	if err != nil {
		return st, err
	}
	st.RemoteHash = agg
	st.InSync = st.LocalHash != "" && st.LocalHash == st.RemoteHash
	return st, nil
}

func (e Engine) Push(ctx context.Context, gameID string, prefer Prefer) error {
	loc, err := detect.FindGame(gameID, e.Cfg.GameOverrides)
	if err != nil {
		return err
	}
	if !loc.Exists {
		return errors.New("local save directory missing")
	}
	local, err := manifest.Build(loc.Game.ID, loc.Game.RemoteKey, loc.SteamID, ToolVersion, loc.Path)
	if err != nil {
		return err
	}

	st, err := e.Status(ctx, gameID)
	if err != nil {
		return err
	}
	if st.HasRemote && st.RemoteHash != "" && st.RemoteHash != local.Aggregate {
		last, _ := e.loadLastSync(loc.Game.ID)
		if last.SyncHash() != "" && local.Aggregate != last.SyncHash() && st.RemoteHash != last.SyncHash() {
			if prefer != PreferLocal {
				return &ConflictError{
					GameID:       loc.Game.ID,
					LocalHash:    local.Aggregate,
					RemoteHash:   st.RemoteHash,
					LastSyncHash: last.SyncHash(),
				}
			}
		}
	}

	if _, err := e.Snap.Capture(loc.Game.ID, loc.Path); err != nil {
		return fmt.Errorf("snapshot before push: %w", err)
	}

	staging, err := os.MkdirTemp("", "ashen-push-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := copyTree(loc.Path, staging); err != nil {
		return err
	}
	if err := local.Write(filepath.Join(staging, manifest.FileName)); err != nil {
		return err
	}
	remoteDir := e.remoteGameDir(loc)
	e.logf("pushing %s -> %s", loc.Path, remoteDir)
	if err := e.Rclone.CopyToRemote(ctx, staging, remoteDir); err != nil {
		return err
	}
	return e.saveLastSync(local)
}

func (e Engine) Pull(ctx context.Context, gameID string, prefer Prefer) error {
	loc, err := detect.FindGame(gameID, e.Cfg.GameOverrides)
	if err != nil {
		return err
	}
	if loc.Path == "" {
		return errors.New("could not resolve local save path")
	}

	tmp, err := os.MkdirTemp("", "ashen-pull-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	remoteDir := e.remoteGameDir(loc)
	e.logf("pulling %s -> staging", remoteDir)
	if err := e.Rclone.CopyFromRemote(ctx, remoteDir, tmp); err != nil {
		return fmt.Errorf("pull remote: %w", err)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("remote save is empty; nothing to pull")
	}

	var remoteHash string
	remoteHash, err = manifest.ContentAggregate(tmp)
	if err != nil {
		return err
	}

	var localHash string
	if loc.Exists {
		local, err := manifest.Build(loc.Game.ID, loc.Game.RemoteKey, loc.SteamID, ToolVersion, loc.Path)
		if err != nil {
			return err
		}
		localHash = local.Aggregate
		if localHash == remoteHash {
			e.logf("already in sync")
			_ = e.saveLastSync(local)
			return nil
		}
		last, _ := e.loadLastSync(loc.Game.ID)
		if last.SyncHash() != "" && localHash != last.SyncHash() && remoteHash != last.SyncHash() {
			if prefer != PreferRemote {
				return &ConflictError{
					GameID:       loc.Game.ID,
					LocalHash:    localHash,
					RemoteHash:   remoteHash,
					LastSyncHash: last.SyncHash(),
				}
			}
		}
		if _, err := e.Snap.Capture(loc.Game.ID, loc.Path); err != nil {
			return fmt.Errorf("snapshot before pull: %w", err)
		}
	} else {
		if err := os.MkdirAll(loc.Path, 0o755); err != nil {
			return err
		}
	}

	// Remove local files then copy remote (excluding we keep staging clean)
	if err := clearDir(loc.Path); err != nil {
		return err
	}
	if err := copyTree(tmp, loc.Path); err != nil {
		return err
	}
	// drop remote manifest from game dir if present (optional keep — keep for transparency)
	local, err := manifest.Build(loc.Game.ID, loc.Game.RemoteKey, loc.SteamID, ToolVersion, loc.Path)
	if err != nil {
		return err
	}
	// Prefer aggregate from content after copy
	_ = os.Remove(filepath.Join(loc.Path, manifest.FileName))
	local, err = manifest.Build(loc.Game.ID, loc.Game.RemoteKey, loc.SteamID, ToolVersion, loc.Path)
	if err != nil {
		return err
	}
	return e.saveLastSync(local)
}

func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o755)
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func IsConflict(err error) (*ConflictError, bool) {
	var c *ConflictError
	if errors.As(err, &c) {
		return c, true
	}
	return nil, false
}

func ResolvePrefer(s string) Prefer {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "local":
		return PreferLocal
	case "remote":
		return PreferRemote
	default:
		return PreferAbort
	}
}
