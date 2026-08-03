package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const FileName = "manifest.json"

type Manifest struct {
	Version    int               `json:"version"`
	GameID     string            `json:"game_id"`
	RemoteKey  string            `json:"remote_key"`
	SteamID    string            `json:"steam_id"`
	Hostname   string            `json:"hostname"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Tool       string            `json:"tool"`
	Files      map[string]string `json:"files"` // rel path -> sha256
	Aggregate  string            `json:"aggregate"`
}

func HashDir(root string) (map[string]string, string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if lower == strings.ToLower(FileName) {
			return nil
		}
		// Only real save slots — ignore mod logs / notes that may live beside saves.
		if !isSaveFile(lower) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		sum, err := hashFile(path)
		if err != nil {
			return err
		}
		files[rel] = sum
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return files, aggregate(files), nil
}

// ContentAggregate hashes on-disk save files in root.
// It never trusts a co-located manifest.json aggregate (which can go stale).
func ContentAggregate(root string) (string, error) {
	_, agg, err := HashDir(root)
	return agg, err
}

func isSaveFile(lowerName string) bool {
	return strings.HasSuffix(lowerName, ".sl2") || strings.HasSuffix(lowerName, ".sl2.bak")
}

// SyncHash is the aggregate used for conflict detection.
// Prefer recomputing from Files with only save slots, so older state that
// included README.md / notes does not false-conflict after HashDir tightened.
func (m Manifest) SyncHash() string {
	if len(m.Files) == 0 {
		return m.Aggregate
	}
	filtered := map[string]string{}
	for rel, sum := range m.Files {
		base := strings.ToLower(filepath.Base(rel))
		if isSaveFile(base) {
			filtered[rel] = sum
		}
	}
	if len(filtered) == 0 {
		return m.Aggregate
	}
	return aggregate(filtered)
}

func Build(gameID, remoteKey, steamID, tool string, root string) (Manifest, error) {
	host, _ := os.Hostname()
	files, agg, err := HashDir(root)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Version:   1,
		GameID:    gameID,
		RemoteKey: remoteKey,
		SteamID:   steamID,
		Hostname:  host,
		UpdatedAt: time.Now().UTC(),
		Tool:      tool,
		Files:     files,
		Aggregate: agg,
	}, nil
}

func (m Manifest) Write(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Read(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func aggregate(files map[string]string) string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(files[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
