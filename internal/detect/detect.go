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

// AutoDetect finds a save folder for game id without using overrides, and
// requires at least one .sl2 save file to be present.
func AutoDetect(id string) (SaveLocation, []string, error) {
	g, ok := ByID(id)
	if !ok {
		return SaveLocation{}, nil, errors.New("unknown game: " + id)
	}
	cands, err := candidatesFor(g)
	if err != nil {
		return SaveLocation{}, nil, err
	}
	// Prefer candidates that actually contain save files.
	var valid []candidate
	for _, c := range cands {
		if files := ListSaveFiles(c.path); len(files) > 0 {
			valid = append(valid, c)
		}
	}
	if len(valid) == 0 {
		return SaveLocation{}, nil, errors.New("no " + g.Name + " save files found on this machine")
	}
	best := pickBest(g, valid)
	files := ListSaveFiles(best.Path)
	return best, files, nil
}

// ListSaveFiles returns .sl2 / .sl2.bak filenames in dir (non-recursive names only at top level + one walk).
func ListSaveFiles(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".sl2") || strings.HasSuffix(lower, ".sl2.bak") {
			files = append(files, name)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

// ValidateSaveDir checks that path is a directory containing at least one .sl2 save.
func ValidateSaveDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	if len(ListSaveFiles(path)) == 0 {
		return errors.New("no .sl2 save files in folder")
	}
	return nil
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

	// Extra Steam libraries (e.g. /mnt/GAMEDRIVE/SteamLibrary) from libraryfolders.vdf
	var libSources []string
	libSources = append(libSources, roots...)
	for _, r := range roots {
		libSources = append(libSources,
			filepath.Join(r, "steamapps", "libraryfolders.vdf"),
			filepath.Join(r, "config", "libraryfolders.vdf"),
		)
	}
	for _, src := range libSources {
		if strings.HasSuffix(src, ".vdf") {
			roots = append(roots, parseLibraryFolderPaths(src)...)
		}
	}

	// Dedup existing
	var out []string
	seen := map[string]bool{}
	for _, r := range roots {
		r = filepath.Clean(r)
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

// parseLibraryFolderPaths reads Steam libraryfolders.vdf and returns library root paths.
func parseLibraryFolderPaths(vdfPath string) []string {
	data, err := os.ReadFile(vdfPath)
	if err != nil {
		return nil
	}
	var paths []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// "path"		"/mnt/GAMEDRIVE/SteamLibrary"
		if !strings.Contains(line, `"path"`) {
			continue
		}
		// last quoted token is the path
		parts := strings.Split(line, `"`)
		if len(parts) < 4 {
			continue
		}
		p := parts[len(parts)-2]
		if p == "path" || p == "" {
			continue
		}
		paths = append(paths, p)
	}
	return paths
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
	if len(name) < 5 {
		return false
	}
	for _, r := range name {
		// Proton save folders are often hex (e.g. 011000011a0eed1a), not only decimal SteamIDs.
		if r >= '0' && r <= '9' {
			continue
		}
		if r >= 'a' && r <= 'f' {
			continue
		}
		if r >= 'A' && r <= 'F' {
			continue
		}
		return false
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
