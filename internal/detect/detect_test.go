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
	if !looksLikeSteamID("011000011a0eed1a") {
		t.Fatal("expected hex proton steam id folder")
	}
	if looksLikeSteamID("DarkSoulsIII") {
		t.Fatal("name should not look like steam id")
	}
	if looksLikeSteamID("ab") {
		t.Fatal("too short")
	}
}

func TestPickBestNewest(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "0110000000000111")
	newDir := filepath.Join(root, "0110000000000222")
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
	if best.SteamID != "0110000000000222" {
		t.Fatalf("want newest steam id 0110000000000222, got %q path=%s", best.SteamID, best.Path)
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

func TestAutoDetectDS3Live(t *testing.T) {
	loc, files, err := AutoDetect("ds3")
	if err != nil {
		t.Skip("no live DS3 save on this machine:", err)
	}
	t.Logf("path=%s files=%v", loc.Path, files)
	if len(files) == 0 {
		t.Fatal("expected save files")
	}
}
