package detect

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLooksLikeSteamID(t *testing.T) {
	if !looksLikeSteamID("76561198000000000") {
		t.Fatal("expected numeric steam id")
	}
	if looksLikeSteamID("DarkSoulsIII") {
		t.Fatal("name should not look like steam id")
	}
}

func TestPickBestNewest(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "111")
	newDir := filepath.Join(root, "222")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldSave := filepath.Join(oldDir, "DS30000.sl2")
	newSave := filepath.Join(newDir, "DS30000.sl2")
	if err := os.WriteFile(oldSave, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(newSave, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := Games[2] // ds3
	cands := steamIDChildren(root, "windows")
	best := pickBest(g, cands)
	if best.SteamID != "222" {
		t.Fatalf("want newest steam id 222, got %q path=%s", best.SteamID, best.Path)
	}
}

func TestFindSavesOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DS30000.sl2"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	locs, err := FindSaves(map[string]string{"ds3": dir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, loc := range locs {
		if loc.Game.ID == "ds3" {
			found = true
			if !loc.Exists || loc.Path != dir {
				t.Fatalf("unexpected loc: %+v", loc)
			}
		}
	}
	if !found {
		t.Fatal("ds3 override not found")
	}
}
