// Package tags holds the docs bundle tag scheme: SemVer 2.0.0 versions
// without a leading "v", the order of builds, full and moving tag names, and
// which moving tags a new build takes over. It does no I/O.
package tags

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Edge is the version, tag and segment of a build of main.
const Edge = "edge"

var (
	reSemVer = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	reNumber = regexp.MustCompile(`^(0|[1-9]\d*)$`)
	reMinor  = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)$`)
)

// Version is a SemVer 2.0.0 version without build metadata.
type Version struct {
	Major, Minor, Patch uint64
	Pre                 []string // dot-separated prerelease identifiers
	raw                 string
}

// ParseVersion parses "4.4.5" or "1.0.0-beta.5". A leading "v" and build
// metadata are refused.
func ParseVersion(s string) (Version, error) {
	m := reSemVer.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("%q is not a SemVer version (MAJOR.MINOR.PATCH[-PRERELEASE], no leading v)", s)
	}
	v := Version{raw: s}
	var err error
	if v.Major, err = strconv.ParseUint(m[1], 10, 64); err != nil {
		return Version{}, fmt.Errorf("%q: %w", s, err)
	}
	if v.Minor, err = strconv.ParseUint(m[2], 10, 64); err != nil {
		return Version{}, fmt.Errorf("%q: %w", s, err)
	}
	if v.Patch, err = strconv.ParseUint(m[3], 10, 64); err != nil {
		return Version{}, fmt.Errorf("%q: %w", s, err)
	}
	if m[4] != "" {
		v.Pre = strings.Split(m[4], ".")
		for _, id := range v.Pre {
			if isNumeric(id) && len(id) > 1 && id[0] == '0' {
				return Version{}, fmt.Errorf("%q: prerelease identifier %q has a leading zero", s, id)
			}
		}
	}
	return v, nil
}

func (v Version) String() string { return v.raw }

// MinorTag is "MAJOR.MINOR", the minor tag and the URL segment.
func (v Version) MinorTag() string { return fmt.Sprintf("%d.%d", v.Major, v.Minor) }

// MajorTag is "MAJOR".
func (v Version) MajorTag() string { return strconv.FormatUint(v.Major, 10) }

// Compare orders versions by SemVer 2.0.0 precedence: -1, 0 or 1.
func Compare(a, b Version) int {
	for _, p := range [][2]uint64{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if c := cmpUint(p[0], p[1]); c != 0 {
			return c
		}
	}
	switch {
	case len(a.Pre) == 0 && len(b.Pre) == 0:
		return 0
	case len(a.Pre) == 0:
		return 1 // a release follows its prereleases
	case len(b.Pre) == 0:
		return -1
	}
	for i := 0; i < len(a.Pre) && i < len(b.Pre); i++ {
		if c := cmpIdent(a.Pre[i], b.Pre[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(a.Pre), len(b.Pre))
}

func cmpIdent(a, b string) int {
	an, bn := isNumeric(a), isNumeric(b)
	switch {
	case an && bn:
		ai, _ := strconv.ParseUint(a, 10, 64)
		bi, _ := strconv.ParseUint(b, 10, 64)
		return cmpUint(ai, bi)
	case an:
		return -1 // numeric identifiers have lower precedence
	case bn:
		return 1
	}
	return strings.Compare(a, b)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func cmpUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpInt(a, b int) int { return cmpUint(uint64(a), uint64(b)) } //nolint:gosec // lengths are never negative

// Build is one pushed manifest, identified by its annotations.
type Build struct {
	Version  Version
	Revision int
	Edge     bool
	Digest   string
}

// NewBuild makes a build from a manifest's version and revision
// annotations ("edge" for a build of main).
func NewBuild(version string, revision int, digest string) (Build, error) {
	if revision < 0 {
		return Build{}, fmt.Errorf("revision %d is negative", revision)
	}
	if version == Edge {
		if revision != 0 {
			return Build{}, fmt.Errorf("an edge build has revision 0, not %d", revision)
		}
		return Build{Edge: true, Digest: digest}, nil
	}
	v, err := ParseVersion(version)
	if err != nil {
		return Build{}, err
	}
	return Build{Version: v, Revision: revision, Digest: digest}, nil
}

// FullTag is "<version>.<revision>", the immutable tag of a release build;
// an edge build has none.
func (b Build) FullTag() string {
	if b.Edge {
		return ""
	}
	return fmt.Sprintf("%s.%d", b.Version, b.Revision)
}

// Segment is the URL segment a build is shown under: "MAJOR.MINOR" or "edge".
func (b Build) Segment() string {
	if b.Edge {
		return Edge
	}
	return b.Version.MinorTag()
}

func (b Build) String() string {
	if b.Edge {
		return Edge
	}
	return b.FullTag()
}

// CompareBuilds orders release builds by version precedence, then by
// revision. Edge builds take no part in the order.
func CompareBuilds(a, b Build) int {
	if c := Compare(a.Version, b.Version); c != 0 {
		return c
	}
	return cmpInt(a.Revision, b.Revision)
}

// Line is one moving tag and the builds it may point at.
type Line struct {
	Tag     string
	Matches func(Build) bool
}

// Lines returns the moving tags of b's line, release first: the release
// tag, the minor tag and the major tag; for an edge build only "edge".
func Lines(b Build) []Line {
	if b.Edge {
		return []Line{{Tag: Edge, Matches: func(o Build) bool { return o.Edge }}}
	}
	v := b.Version
	return []Line{
		{Tag: v.String(), Matches: func(o Build) bool { return !o.Edge && Compare(o.Version, v) == 0 }},
		{Tag: v.MinorTag(), Matches: func(o Build) bool { return !o.Edge && o.Version.Major == v.Major && o.Version.Minor == v.Minor }},
		{Tag: v.MajorTag(), Matches: func(o Build) bool { return !o.Edge && o.Version.Major == v.Major }},
	}
}

// Promotion says which moving tags move to d, given every release build of
// the repository (d included or not): a tag moves only when no build of
// its line is newer than d. An edge build always takes "edge".
func Promotion(d Build, builds []Build) []string {
	var out []string
	for _, l := range Lines(d) {
		if d.Edge {
			out = append(out, l.Tag)
			continue
		}
		newest := true
		for _, o := range builds {
			if l.Matches(o) && CompareBuilds(o, d) > 0 {
				newest = false
				break
			}
		}
		if newest {
			out = append(out, l.Tag)
		}
	}
	return out
}

// NextRevision is the revision a docs revision of v takes: one more than
// the highest published, refused when revision 0 does not exist.
func NextRevision(v Version, builds []Build) (int, error) {
	highest := -1
	for _, b := range builds {
		if !b.Edge && Compare(b.Version, v) == 0 && b.Revision > highest {
			highest = b.Revision
		}
	}
	if highest < 0 {
		return 0, fmt.Errorf("%s has no published build (%s.0); a docs revision needs the release published first", v, v)
	}
	return highest + 1, nil
}

// IsMinorTag reports whether t is a minor tag, "MAJOR.MINOR".
func IsMinorTag(t string) bool { return reMinor.MatchString(t) }

// ParseMinor splits a minor tag into its numbers.
func ParseMinor(t string) (major, minor uint64, err error) {
	m := reMinor.FindStringSubmatch(t)
	if m == nil {
		return 0, 0, fmt.Errorf("%q is not MAJOR.MINOR", t)
	}
	major, _ = strconv.ParseUint(m[1], 10, 64)
	minor, _ = strconv.ParseUint(m[2], 10, 64)
	return major, minor, nil
}

// CompareMinor orders two "MAJOR.MINOR" segments numerically; "edge" sorts
// after every minor.
func CompareMinor(a, b string) int {
	if a == b {
		return 0
	}
	if a == Edge {
		return 1
	}
	if b == Edge {
		return -1
	}
	am, an, _ := ParseMinor(a)
	bm, bn, _ := ParseMinor(b)
	if c := cmpUint(am, bm); c != 0 {
		return c
	}
	return cmpUint(an, bn)
}

// IsSignatureTag reports a cosign fallback tag, "sha256-<hex>", which names
// no build.
func IsSignatureTag(t string) bool { return strings.HasPrefix(t, "sha256-") }

// SplitFullTag splits a tag that has the form of a full tag into its
// version and revision. Whether the tag really names that build is decided
// by the manifest's annotations, never by the name.
func SplitFullTag(t string) (version string, revision int, ok bool) {
	i := strings.LastIndexByte(t, '.')
	if i < 0 || !reNumber.MatchString(t[i+1:]) {
		return "", 0, false
	}
	if _, err := ParseVersion(t[:i]); err != nil {
		return "", 0, false
	}
	r, err := strconv.Atoi(t[i+1:])
	if err != nil {
		return "", 0, false
	}
	return t[:i], r, true
}
