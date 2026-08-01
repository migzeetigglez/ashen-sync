package detect

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type SaveLocation struct {
	Game     Game
	Path     string
	SteamID  string
	Exists   bool
	ModTime  time.Time
	Platform string // windows | proton
}

// FindSaves discovers save directories for known games.
func FindSaves(overrides map[string]string) ([]SaveLocation, error) {
	var out []SaveLocation
	for _, g := range Games {
		if override, ok := overrides[g.ID]; ok && override != "" {
			info, err := os.Stat(override)
			loc := SaveLocation{
				Game:     g,
				Path:     override,
				SteamID:  filepath.Base(override),
				Exists:   err == nil && info.IsDir(),
				Platform: "override",
			}
			if loc.Exists {
				loc.ModTime = newestModTime(override)
			}
			out = append(out, loc)
			continue
		}

		candidates, err := candidatesFor(g)
		if err != nil {
			return nil, err
		}
		best := pickBest(g, candidates)
		if best.Path != "" {
			out = append(out, best)
		}
	}
	return out, nil
}

func FindGame(id string, overrides map[string]string) (SaveLocation, error) {
	g, ok := ByID(id)
	if !ok {
		return SaveLocation{}, errors.New("unknown game: " + id)
	}
	all, err := FindSaves(overrides)
	if err != nil {
		return SaveLocation{}, err
	}
	for _, loc := range all {
		if loc.Game.ID == g.ID {
			return loc, nil
		}
	}
	return SaveLocation{}, errors.New("no save directory found for " + g.Name)
}

type candidate struct {
	path     string
	steamID  string
	platform string
	modTime  time.Time
}

func candidatesFor(g Game) ([]candidate, error) {
	var cands []candidate

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			root := filepath.Join(appData, g.FolderName)
			cands = append(cands, steamIDChildren(root, "windows")...)
		}
	}

	// Proton prefixes (Linux / Steam Deck; also useful if somehow present)
	for _, steamRoot := range steamRoots() {
		pfx := filepath.Join(steamRoot, "steamapps", "compatdata", g.AppID,
			"pfx", "drive_c", "users", "steamuser", "AppData", "Roaming", g.FolderName)
		cands = append(cands, steamIDChildren(pfx, "proton")...)
	}

	return cands, nil
}

func steamRoots() []string {
	var roots []string
	home, err := os.UserHomeDir()
	if err == nil {
		roots = append(roots,
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
		)
	}
	if runtime.GOOS == "darwin" {
		if err == nil {
			roots = append(roots, filepath.Join(home, "Library", "Application Support", "Steam"))
		}
	}
	// Dedup existing
	var out []string
	seen := map[string]bool{}
	for _, r := range roots {
		if seen[r] {
			continue
		}
		seen[r] = true
		if st, err := os.Stat(r); err == nil && st.IsDir() {
			out = append(out, r)
		}
	}
	return out
}

func steamIDChildren(root, platform string) []candidate {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var cands []candidate
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !looksLikeSteamID(name) {
			continue
		}
		p := filepath.Join(root, name)
		cands = append(cands, candidate{
			path:     p,
			steamID:  name,
			platform: platform,
			modTime:  newestModTime(p),
		})
	}
	return cands
}

func looksLikeSteamID(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func newestModTime(dir string) time.Time {
	var newest time.Time
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".sl2") && !strings.HasSuffix(name, ".sl2.bak") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if newest.IsZero() {
		if info, err := os.Stat(dir); err == nil {
			return info.ModTime()
		}
	}
	return newest
}

func pickBest(g Game, cands []candidate) SaveLocation {
	if len(cands) == 0 {
		return SaveLocation{Game: g}
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].modTime.After(cands[j].modTime)
	})
	best := cands[0]
	return SaveLocation{
		Game:     g,
		Path:     best.path,
		SteamID:  best.steamID,
		Exists:   true,
		ModTime:  best.modTime,
		Platform: best.platform,
	}
}
