package update

import "testing"

func TestNormalizeTag(t *testing.T) {
	if got := NormalizeTag(" v1.2.3 "); got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeTag("0.1.0"); got != "0.1.0" {
		t.Fatalf("got %q", got)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.0", "0.1.0", 0},
		{"v0.1.0", "0.1.0", 0},
		{"0.1.0", "0.2.0", -1},
		{"0.2.0", "0.1.0", 1},
		{"0.1.0", "0.1.1", -1},
		{"1.0", "1.0.0", 0},
		{"1.0.0", "1.0.0-beta", 0},
		{"0.9.9", "0.10.0", -1},
	}
	for _, tc := range cases {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestAssetNameForGOOS(t *testing.T) {
	w, err := AssetNameForGOOS("windows")
	if err != nil || w != "save-channeler-windows-amd64.zip" {
		t.Fatalf("windows: %q %v", w, err)
	}
	l, err := AssetNameForGOOS("linux")
	if err != nil || l != "save-channeler-linux-amd64.tar.gz" {
		t.Fatalf("linux: %q %v", l, err)
	}
	if _, err := AssetNameForGOOS("darwin"); err == nil {
		t.Fatal("expected error for darwin")
	}
}
