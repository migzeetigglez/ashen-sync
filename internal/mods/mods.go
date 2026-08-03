package mods

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Entry is one mod listed in a save-folder mod log.
type Entry struct {
	Name string
	URL  string // optional
}

var (
	mdLink  = regexp.MustCompile(`^\s*[-*+]\s+\[([^\]]+)\]\(([^)]+)\)\s*$`)
	mdBold  = regexp.MustCompile(`^\s*[-*+]\s+\*\*([^*]+)\*\*(?:\s*[—:-]\s*(https?://\S+))?\s*$`)
	mdPlain = regexp.MustCompile(`^\s*[-*+]\s+(.+?)\s*$`)
	urlOnly = regexp.MustCompile(`(?i)https?://\S+`)
)

// Candidates are filenames looked for inside a save directory.
var Candidates = []string{
	"README.md",
	"readme.md",
	"MODS.md",
	"mods.md",
}

// FindLogPath returns the first mod-log file in saveDir, if any.
func FindLogPath(saveDir string) string {
	if saveDir == "" {
		return ""
	}
	for _, name := range Candidates {
		p := filepath.Join(saveDir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// Load reads the mod log for a save directory.
func Load(saveDir string) (path string, entries []Entry, err error) {
	path = FindLogPath(saveDir)
	if path == "" {
		return "", nil, nil
	}
	entries, err = ParseFile(path)
	return path, entries, err
}

// ParseFile parses a markdown mod list from path.
func ParseFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse extracts mod entries from markdown list items.
// Preferred form:
//
//	# Mods
//	- [Mod Name](https://nexusmods.com/...)
//
// Also accepts:
//
//	- **Mod Name** — https://...
//	- Mod Name
//
// If the file has headings, only sections whose title contains "mod" are read.
// If there are no headings, all list items are treated as mods.
func Parse(r io.Reader) ([]Entry, error) {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	var out []Entry
	inModsSection := false
	sawHeading := false
	inHTMLComment := false

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Strip / skip HTML comments (including multi-line <!-- ... -->).
		if inHTMLComment {
			if i := strings.Index(trimmed, "-->"); i >= 0 {
				inHTMLComment = false
				trimmed = strings.TrimSpace(trimmed[i+3:])
				if trimmed == "" {
					continue
				}
			} else {
				continue
			}
		}
		for {
			start := strings.Index(trimmed, "<!--")
			if start < 0 {
				break
			}
			rest := trimmed[start+4:]
			if end := strings.Index(rest, "-->"); end >= 0 {
				trimmed = strings.TrimSpace(trimmed[:start] + rest[end+3:])
				continue
			}
			trimmed = strings.TrimSpace(trimmed[:start])
			inHTMLComment = true
			break
		}
		if trimmed == "" {
			continue
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "#") {
			sawHeading = true
			title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			titleLower := strings.ToLower(title)
			inModsSection = strings.Contains(titleLower, "mod")
			continue
		}

		if sawHeading && !inModsSection {
			continue
		}

		if m := mdLink.FindStringSubmatch(trimmed); m != nil {
			out = append(out, Entry{Name: strings.TrimSpace(m[1]), URL: strings.TrimSpace(m[2])})
			continue
		}
		if m := mdBold.FindStringSubmatch(trimmed); m != nil {
			e := Entry{Name: strings.TrimSpace(m[1])}
			if len(m) > 2 {
				e.URL = strings.TrimSpace(m[2])
			}
			out = append(out, e)
			continue
		}
		if m := mdPlain.FindStringSubmatch(trimmed); m != nil {
			name := strings.TrimSpace(m[1])
			if strings.HasPrefix(name, "[") {
				continue
			}
			e := Entry{Name: name}
			if u := urlOnly.FindString(name); u != "" {
				e.URL = u
				e.Name = strings.TrimSpace(strings.Replace(name, u, "", 1))
				e.Name = strings.Trim(e.Name, " —-:|")
				if e.Name == "" {
					e.Name = u
				}
			}
			out = append(out, e)
		}
	}
	if err := sc.Err(); err != nil {
		return out, err
	}
	return out, nil
}
