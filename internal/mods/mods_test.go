package mods

import (
	"strings"
	"testing"
)

func TestParseMarkdownLinks(t *testing.T) {
	src := `# Mods

- [Dark Souls Re-Remastered](https://www.nexusmods.com/darksoulsremastered/mods/642)
- [DS-R Material Update](https://www.nexusmods.com/darksoulsremastered/mods/90)

## Notes
- ignore me
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 mods, got %d: %+v", len(entries), entries)
	}
	if entries[0].Name != "Dark Souls Re-Remastered" {
		t.Fatalf("name: %q", entries[0].Name)
	}
	if !strings.Contains(entries[0].URL, "mods/642") {
		t.Fatalf("url: %q", entries[0].URL)
	}
}

func TestParseNoHeading(t *testing.T) {
	src := `
- **Cool Mod** — https://example.com/mod
- Plain Mod Name
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2, got %d %+v", len(entries), entries)
	}
	if entries[0].URL != "https://example.com/mod" {
		t.Fatalf("url %q", entries[0].URL)
	}
	if entries[1].Name != "Plain Mod Name" {
		t.Fatalf("name %q", entries[1].Name)
	}
}

func TestParseSkipsHTMLComments(t *testing.T) {
	src := `# Mods

<!-- Example only:
- [Mod Name](https://www.nexusmods.com/darksouls3/mods/123)
-->
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("want 0 mods inside HTML comments, got %+v", entries)
	}
}
