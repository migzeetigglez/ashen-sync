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
