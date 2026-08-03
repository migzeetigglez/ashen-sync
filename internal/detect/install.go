package detect

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FindExeDir locates the directory that contains the game's Windows executable
// (e.g. …/DARK SOULS III/Game), searching Steam libraries via appmanifests.
func FindExeDir(g Game) string {
	return findExeDirInRoots(g, steamRoots())
}

func findExeDirInRoots(g Game, roots []string) string {
	if g.ExeName == "" || g.AppID == "" {
		return ""
	}
	for _, root := range roots {
		acf := filepath.Join(root, "steamapps", "appmanifest_"+g.AppID+".acf")
		installdir := parseACFInstallDir(acf)
		if installdir == "" {
			continue
		}
		common := filepath.Join(root, "steamapps", "common", installdir)
		if dir := findExeDirUnder(common, g.ExeName); dir != "" {
			return dir
		}
	}
	return ""
}

// parseACFInstallDir reads "installdir" from a Steam appmanifest_*.acf file.
func parseACFInstallDir(acfPath string) string {
	data, err := os.ReadFile(acfPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, `"installdir"`) {
			continue
		}
		parts := strings.Split(line, `"`)
		if len(parts) < 4 {
			continue
		}
		p := parts[len(parts)-2]
		if p == "installdir" || p == "" {
			continue
		}
		return p
	}
	return ""
}

// findExeDirUnder walks root looking for exeName; returns that file's directory.
func findExeDirUnder(root, exeName string) string {
	want := strings.ToLower(exeName)
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.ToLower(d.Name()) == want {
			found = filepath.Dir(path)
			return fs.SkipAll
		}
		return nil
	})
	return found
}
