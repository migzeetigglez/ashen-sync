package update

import (
	"strconv"
	"strings"
)

// NormalizeTag strips a leading "v" and trims space.
func NormalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	return strings.TrimPrefix(tag, "v")
}

// CompareVersions returns -1 if a < b, 0 if equal, 1 if a > b.
// Uses numeric dot segments; non-numeric suffixes are ignored for v1.
func CompareVersions(a, b string) int {
	as := versionParts(NormalizeTag(a))
	bs := versionParts(NormalizeTag(b))
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func versionParts(v string) []int {
	if v == "" {
		return []int{0}
	}
	// Drop prerelease / build metadata.
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	segs := strings.Split(v, ".")
	out := make([]int, 0, len(segs))
	for _, s := range segs {
		n, err := strconv.Atoi(s)
		if err != nil {
			// Stop at first non-numeric segment.
			break
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int{0}
	}
	return out
}
