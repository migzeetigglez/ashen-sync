package mods

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// installSignature matches known DLL / companion files next to the game exe.
type installSignature struct {
	Name  string
	URL   string
	Files []string // any of these present (case-insensitive basename)
	// Paths are relative to the game exe dir; any existing path matches.
	// Use for mods living under ModEngine folders (e.g. mods/visual_overhaul/...).
	Paths []string
}

var signaturesByGame = map[string][]installSignature{
	"ds3": {
		{
			Name:  "Honest Merchant",
			URL:   "https://www.nexusmods.com/darksouls3/mods/607",
			Files: []string{"honestMerchant.ini", "HonestMerchant-README.txt", "HonestMerchant.dll"},
		},
		{
			Name:  "Obanon's Incandescent ReShade",
			URL:   "https://www.nexusmods.com/darksouls3/mods/12",
			Files: []string{"Obanons Incandescent ReShade 2.2.ini", "Obanons Incandescent ReShade 2.0.ini"},
		},
		{
			Name:  "DS3 TAA",
			URL:   "https://www.nexusmods.com/darksouls3/mods/1849",
			Files: []string{"DS3_TAA.ini"},
		},
		{
			Name: "DS3 Visual Overhaul",
			URL:  "https://www.nexusmods.com/darksouls3/mods/1849",
			Paths: []string{
				"mods/visual_overhaul/param/drawparam/m30_00_0000.gparam.dcx",
				"mods/visual_overhaul/map/MapStudio/m30_00_00_00.msb.dcx",
			},
		},
		{
			Name:  "ModEngine2",
			URL:   "https://www.nexusmods.com/darksoulsremastered/mods/766",
			Files: []string{"modengine.toml", "lazyLoad.ini"},
		},
		{
			Name:  "ReShade",
			URL:   "https://reshade.me/",
			Files: []string{"ReShade.ini", "dxgi.dll", "d3d11.dll"},
		},
		{
			Name:  "Blue Sentinel",
			URL:   "https://www.nexusmods.com/darksouls3/mods/723",
			Files: []string{"BlueSentinel.dll", "blue_sentinel.dll"},
		},
		{
			Name:  "Wex Dust",
			URL:   "https://www.nexusmods.com/darksouls3/mods/462",
			Files: []string{"wexdust.dll", "WexDust.dll", "wex_dust.ini"},
		},
	},
	"ds2": {
		{
			Name:  "Blue Sentinel",
			URL:   "https://www.nexusmods.com/darksouls2/mods/1294",
			Files: []string{"BlueSentinel.dll", "blue_sentinel.dll"},
		},
	},
	"ds2-vanilla": {
		{
			Name:  "Blue Sentinel",
			URL:   "https://www.nexusmods.com/darksouls2/mods/1294",
			Files: []string{"BlueSentinel.dll", "blue_sentinel.dll"},
		},
	},
}

// DetectInstall scans the game executable directory for known mods (and a
// generic dinput8.dll fallback when nothing else matches).
func DetectInstall(gameID, exeDir string) []Entry {
	if exeDir == "" {
		return nil
	}
	names, err := listBasenames(exeDir)
	if err != nil {
		return nil
	}

	var out []Entry
	matched := map[string]bool{}
	claimedFiles := map[string]bool{}

	for _, sig := range signaturesByGame[gameID] {
		filesHit := len(sig.Files) > 0 && anyFilePresent(names, sig.Files...)
		pathsHit := len(sig.Paths) > 0 && anyPathPresent(exeDir, sig.Paths...)
		if !filesHit && !pathsHit {
			continue
		}
		out = append(out, Entry{Name: sig.Name, URL: sig.URL})
		matched[strings.ToLower(sig.Name)] = true
		for _, f := range sig.Files {
			if actual, ok := names[strings.ToLower(f)]; ok {
				claimedFiles[strings.ToLower(actual)] = true
			}
		}
	}

	// Honest Merchant chainload: dll=other.dll in honestMerchant.ini
	if iniPath := findNamed(exeDir, names, "honestMerchant.ini"); iniPath != "" {
		for _, e := range chainloadEntries(iniPath, names, claimedFiles) {
			if matched[strings.ToLower(e.Name)] {
				continue
			}
			out = append(out, e)
			matched[strings.ToLower(e.Name)] = true
		}
	}

	hasDinput8 := anyFilePresent(names, "dinput8.dll")
	if hasDinput8 && len(out) == 0 {
		out = append(out, Entry{Name: "dinput8.dll mod (unrecognized)"})
	}

	return out
}

// Merge combines auto-detected install mods with save-folder log entries.
// Auto entries come first; later entries are skipped only when the
// case-insensitive name already appears (same Nexus page can host multiple
// distinct mods, e.g. DS3 TAA and Visual Overhaul).
func Merge(auto, log []Entry) []Entry {
	var out []Entry
	seenName := map[string]bool{}

	add := func(e Entry) {
		if e.Name == "" && e.URL == "" {
			return
		}
		n := strings.ToLower(strings.TrimSpace(e.Name))
		if n == "" {
			n = strings.ToLower(strings.TrimSpace(e.URL))
		}
		if seenName[n] {
			return
		}
		seenName[n] = true
		out = append(out, e)
	}

	for _, e := range auto {
		add(e)
	}
	for _, e := range log {
		add(e)
	}
	return out
}

func listBasenames(dir string) (map[string]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// lower -> actual name
	out := make(map[string]string, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		out[strings.ToLower(e.Name())] = e.Name()
	}
	return out, nil
}

func anyFilePresent(names map[string]string, files ...string) bool {
	for _, f := range files {
		if _, ok := names[strings.ToLower(f)]; ok {
			return true
		}
	}
	return false
}

func anyPathPresent(exeDir string, rels ...string) bool {
	for _, rel := range rels {
		if rel == "" {
			continue
		}
		if st, err := os.Stat(filepath.Join(exeDir, filepath.FromSlash(rel))); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func findNamed(dir string, names map[string]string, want string) string {
	actual, ok := names[strings.ToLower(want)]
	if !ok {
		return ""
	}
	return filepath.Join(dir, actual)
}

func chainloadEntries(iniPath string, names map[string]string, claimedFiles map[string]bool) []Entry {
	f, err := os.Open(iniPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []Entry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "dll=") {
			continue
		}
		dll := strings.TrimSpace(line[4:])
		dll = strings.Trim(dll, `"'`)
		if dll == "" {
			continue
		}
		base := filepath.Base(dll)
		if claimedFiles[strings.ToLower(base)] {
			continue
		}
		if !anyFilePresent(names, base) {
			continue
		}
		name := strings.TrimSuffix(base, filepath.Ext(base))
		if name == "" {
			name = base
		}
		out = append(out, Entry{Name: name + " (chainloaded)"})
		claimedFiles[strings.ToLower(base)] = true
	}
	return out
}
