package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindExeDirInRoots(t *testing.T) {
	root := t.TempDir()
	acfDir := filepath.Join(root, "steamapps")
	gameDir := filepath.Join(root, "steamapps", "common", "DARK SOULS III", "Game")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	acf := `"AppState"
{
	"appid"		"374320"
	"installdir"		"DARK SOULS III"
}
`
	if err := os.WriteFile(filepath.Join(acfDir, "appmanifest_374320.acf"), []byte(acf), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, "DarkSoulsIII.exe"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, ok := ByID("ds3")
	if !ok {
		t.Fatal("ds3")
	}
	got := findExeDirInRoots(g, []string{root})
	if got != gameDir {
		t.Fatalf("want %q, got %q", gameDir, got)
	}
}

func TestFindExeDirLiveDS3(t *testing.T) {
	g, ok := ByID("ds3")
	if !ok {
		t.Fatal("ds3")
	}
	dir := FindExeDir(g)
	if dir == "" {
		t.Skip("DS3 install not found on this machine")
	}
	exe := filepath.Join(dir, g.ExeName)
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("exe missing under %s: %v", dir, err)
	}
	t.Logf("exe dir=%s", dir)
}
