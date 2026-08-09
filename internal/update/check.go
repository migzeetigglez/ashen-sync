package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/migzeetigglez/ashen-sync/internal/version"
)

const (
	defaultRepoOwner = "migzeetigglez"
	defaultRepoName  = "ashen-sync"
	releasesURLFmt   = "https://api.github.com/repos/%s/%s/releases/latest"
	userAgent        = "save-channeler/" + "updater"
)

// Info describes a newer GitHub release when Available is true.
type Info struct {
	Available bool
	Tag       string // e.g. v0.2.0
	Version   string // e.g. 0.2.0
	Body      string
	AssetName string
	AssetURL  string
	Current   string
	HTMLURL   string
}

type releaseJSON struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Checker looks up the latest GitHub release.
type Checker struct {
	Owner      string
	Repo       string
	BaseURL    string // override API root for tests; empty = GitHub
	HTTPClient *http.Client
	GOOS       string // override for tests; empty = runtime
	LocalVer   string // override for tests; empty = version.Version
}

func (c Checker) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c Checker) owner() string {
	if c.Owner != "" {
		return c.Owner
	}
	return defaultRepoOwner
}

func (c Checker) repo() string {
	if c.Repo != "" {
		return c.Repo
	}
	return defaultRepoName
}

// Check returns update info for the latest GitHub release vs the local version.
func Check(ctx context.Context) (Info, error) {
	return Checker{}.Check(ctx)
}

// Check implements Checker.
func (c Checker) Check(ctx context.Context) (Info, error) {
	local := c.LocalVer
	if local == "" {
		local = version.Version
	}
	goos := c.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	assetWant, assetErr := AssetNameForGOOS(goos)

	url := fmt.Sprintf(releasesURLFmt, c.owner(), c.repo())
	if c.BaseURL != "" {
		url = strings.TrimRight(c.BaseURL, "/") + "/repos/" + c.owner() + "/" + c.repo() + "/releases/latest"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent+"/"+NormalizeTag(local))

	res, err := c.client().Do(req)
	if err != nil {
		return Info{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return Info{}, err
	}
	if res.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("github releases: HTTP %d: %s", res.StatusCode, truncate(string(body), 200))
	}

	var rel releaseJSON
	if err := json.Unmarshal(body, &rel); err != nil {
		return Info{}, fmt.Errorf("decode release: %w", err)
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return Info{}, fmt.Errorf("release missing tag_name")
	}

	info := Info{
		Tag:     rel.TagName,
		Version: NormalizeTag(rel.TagName),
		Body:    strings.TrimSpace(rel.Body),
		Current: NormalizeTag(local),
		HTMLURL: rel.HTMLURL,
	}
	if assetWant != "" {
		for _, a := range rel.Assets {
			if a.Name == assetWant {
				info.AssetName = a.Name
				info.AssetURL = a.BrowserDownloadURL
				break
			}
		}
		if info.AssetURL == "" {
			return info, fmt.Errorf("release %s has no asset %q", rel.TagName, assetWant)
		}
	} else if assetErr != nil {
		// Unsupported GOOS for in-app install (e.g. macOS) — still report version.
		_ = assetErr
	}
	if CompareVersions(info.Version, info.Current) > 0 {
		info.Available = true
	}
	return info, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
