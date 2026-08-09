package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckAvailable(t *testing.T) {
	const payload = `{
	  "tag_name": "v0.2.0",
	  "body": "## What's new\n- updater",
	  "html_url": "https://github.com/migzeetigglez/ashen-sync/releases/tag/v0.2.0",
	  "assets": [
	    {"name": "save-channeler-linux-amd64.tar.gz", "browser_download_url": "https://example.com/linux.tgz"},
	    {"name": "save-channeler-windows-amd64.zip", "browser_download_url": "https://example.com/win.zip"}
	  ]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := Checker{
		BaseURL:    srv.URL,
		GOOS:       "linux",
		LocalVer:   "0.1.0",
		HTTPClient: srv.Client(),
	}
	info, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available {
		t.Fatalf("expected available, got %+v", info)
	}
	if info.Version != "0.2.0" || info.AssetURL != "https://example.com/linux.tgz" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if !strings.Contains(info.Body, "updater") {
		t.Fatalf("body: %q", info.Body)
	}
}

func TestCheckNotNewer(t *testing.T) {
	const payload = `{
	  "tag_name": "v0.1.0",
	  "body": "",
	  "html_url": "https://example.com",
	  "assets": [
	    {"name": "save-channeler-windows-amd64.zip", "browser_download_url": "https://example.com/win.zip"}
	  ]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := Checker{
		BaseURL:    srv.URL,
		GOOS:       "windows",
		LocalVer:   "0.1.0",
		HTTPClient: srv.Client(),
	}
	info, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Available {
		t.Fatalf("should not be available: %+v", info)
	}
}
