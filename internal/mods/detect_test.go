package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstallHonestMerchant(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "dinput8.dll"), "dll")
	mustWrite(t, filepath.Join(dir, "honestMerchant.ini"), "[HONESTMERCHANT]\n")

	got := DetectInstall("ds3", dir)
	if len(got) != 1 || got[0].Name != "Honest Merchant" {
		t.Fatalf("got %+v", got)
	}
	if got[0].URL == "" {
		t.Fatal("expected nexus URL")
	}
}

func TestDetectInstallUnrecognizedDinput8(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "dinput8.dll"), "dll")

	got := DetectInstall("ds3", dir)
	if len(got) != 1 || got[0].Name != "dinput8.dll mod (unrecognized)" {
		t.Fatalf("got %+v", got)
	}
}

func TestDetectInstallVisualOverhaulPath(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "mods", "visual_overhaul", "param", "drawparam", "m30_00_0000.gparam.dcx")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, marker, "gparam")

	got := DetectInstall("ds3", dir)
	found := false
	for _, e := range got {
		if e.Name == "DS3 Visual Overhaul" {
			found = true
			if e.URL == "" {
				t.Fatal("expected nexus URL")
			}
		}
	}
	if !found {
		t.Fatalf("missing Visual Overhaul: %+v", got)
	}
}

func TestDetectInstallChainload(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "dinput8.dll"), "dll")
	mustWrite(t, filepath.Join(dir, "coolmod.dll"), "dll")
	mustWrite(t, filepath.Join(dir, "honestMerchant.ini"), "[CHAINLOAD]\ndll=coolmod.dll\n")

	got := DetectInstall("ds3", dir)
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	if !names["Honest Merchant"] {
		t.Fatalf("missing Honest Merchant: %+v", got)
	}
	if !names["coolmod (chainloaded)"] {
		t.Fatalf("missing chainload entry: %+v", got)
	}
}

func TestMergeDedupesByNameKeepsSharedURL(t *testing.T) {
	auto := []Entry{
		{Name: "Honest Merchant", URL: "https://www.nexusmods.com/darksouls3/mods/607"},
		{Name: "DS3 TAA", URL: "https://www.nexusmods.com/darksouls3/mods/1849"},
		{Name: "DS3 Visual Overhaul", URL: "https://www.nexusmods.com/darksouls3/mods/1849"},
	}
	log := []Entry{
		{Name: "Honest Merchant", URL: "https://www.nexusmods.com/darksouls3/mods/607"},
		{Name: "Cool Texture Pack"},
	}
	got := Merge(auto, log)
	if len(got) != 4 {
		t.Fatalf("want 4, got %+v", got)
	}
	if got[0].Name != "Honest Merchant" || got[1].Name != "DS3 TAA" || got[2].Name != "DS3 Visual Overhaul" || got[3].Name != "Cool Texture Pack" {
		t.Fatalf("order/names: %+v", got)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
