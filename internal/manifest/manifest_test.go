package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashDirStable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.sl2"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.sl2"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, agg1, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, agg2, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if agg1 != agg2 || agg1 == "" {
		t.Fatalf("aggregate unstable: %s vs %s", agg1, agg2)
	}

	m, err := Build("ds3", "dark-souls-iii", "123", "test", dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	if err := m.Write(path); err != nil {
		t.Fatal(err)
	}
	m2, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Aggregate != m.Aggregate {
		t.Fatal("round trip mismatch")
	}
}

func TestHashDirIgnoresModLogAndManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DS30000.sl2"), []byte("save"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, aggSaveOnly, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Mods\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"aggregate":"stale"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	files, aggWithExtras, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if aggSaveOnly != aggWithExtras {
		t.Fatalf("non-save files changed aggregate: %s vs %s (files=%v)", aggSaveOnly, aggWithExtras, files)
	}
	if _, ok := files["README.md"]; ok {
		t.Fatal("README.md should not be hashed")
	}
	if _, ok := files[FileName]; ok {
		t.Fatal("manifest.json should not be hashed")
	}
}

func TestHashDirIgnoresStaleManifestClaim(t *testing.T) {
	// Content hash must reflect on-disk saves, not a lying manifest.json.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DS30000.sl2"), []byte("actual-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, want, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	stale := Manifest{
		Version:   1,
		GameID:    "ds3",
		Files:     map[string]string{"DS30000.sl2": "deadbeef"},
		Aggregate: "this-is-not-the-content-hash",
	}
	if err := stale.Write(filepath.Join(dir, FileName)); err != nil {
		t.Fatal(err)
	}
	got, err := ContentAggregate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ContentAggregate followed stale manifest: got %s want %s", got, want)
	}
	if got == stale.Aggregate {
		t.Fatal("ContentAggregate must not return manifest.Aggregate")
	}
}

func TestSyncHashStripsNonSaveFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DS30000.sl2"), []byte("save"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, want, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := Manifest{
		Files: map[string]string{
			"DS30000.sl2": "abc",
			"README.md":   "def",
		},
		Aggregate: "old-aggregate-with-readme",
	}
	// SyncHash should match hashing only the save entry values we stored…
	onlySave := Manifest{Files: map[string]string{"DS30000.sl2": "abc"}}
	if m.SyncHash() != onlySave.SyncHash() {
		t.Fatalf("SyncHash did not drop README: %s", m.SyncHash())
	}
	if m.SyncHash() == m.Aggregate {
		t.Fatal("SyncHash should not equal polluted Aggregate")
	}
	// And live ContentAggregate ignores README entirely:
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("mods"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ContentAggregate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ContentAggregate changed after README: %s vs %s", got, want)
	}
}
