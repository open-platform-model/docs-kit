package dialect

import (
	"regexp"
	"strings"
)

var (
	reExternal    = regexp.MustCompile(`^(https?:|mailto:|#)`)
	reDocs        = regexp.MustCompile(`^/docs/`)
	reDocsOK      = regexp.MustCompile(`^/docs/([a-z0-9-]+/)*(#[^ ]*)?$`)
	reEnh         = regexp.MustCompile(`^/enhancements([/#]|$)`)
	reEnhOK       = regexp.MustCompile(`^/enhancements/([0-9][0-9][0-9][0-9]/((problem|design|decisions|graduation|risks|operational|questions)/)?)?(#[^ ]*)?$`)
	reCatalogs    = regexp.MustCompile(`^/catalogs([/#]|$)`)
	reCatalogLink = regexp.MustCompile(`^/catalogs/([a-z0-9]+(?:-[a-z0-9]+)*)/(?:((?:0|[1-9][0-9]*)(?:\.(?:0|[1-9][0-9]*))?|edge)/((?:[a-z0-9-]+/)*))?(#[^ ]*)?$`)
)

// dest checks one link destination.
func (p *pageLint) dest(t string) {
	switch {
	case reExternal.MatchString(t):
	case reDocs.MatchString(t):
		if !reDocsOK.MatchString(t) {
			p.err(p.nr, `internal link "`+t+`": write /docs/<section>/<page>/ with a trailing slash`)
		}
	case reEnh.MatchString(t):
		if !reEnhOK.MatchString(t) {
			p.err(p.nr, `enhancement link "`+t+`": write /enhancements/, /enhancements/<NNNN>/ or /enhancements/<NNNN>/<document>/ with a trailing slash`)
		}
	case reCatalogs.MatchString(t):
		p.catalogLink(t)
	default:
		p.err(p.nr, `link "`+t+`": internal links are root-absolute /docs/<section>/<page>/ or /enhancements/<NNNN>/ (no relative, .md or version-prefixed links)`)
	}
}

// catalogLink checks a /catalogs/ link: its form, and the segment rules of
// the lint's mode.
func (p *pageLint) catalogLink(t string) {
	m := reCatalogLink.FindStringSubmatch(t)
	if m == nil {
		p.err(p.nr, `catalog link "`+t+`": write /catalogs/<name>/ or /catalogs/<name>/<MAJOR>/<path>/ with a trailing slash`)
		return
	}
	name, segment, path := m[1], m[2], m[3]
	root := "/catalogs/" + name + "/"
	if p.opts.Mode == Bundle && root == p.opts.Bundle.Root {
		p.ownLink(t, segment, path)
		return
	}
	if segment == "" || isMajor(segment) {
		return
	}
	major := "<MAJOR>"
	if segment != "edge" {
		major, _, _ = strings.Cut(segment, ".")
	}
	if p.opts.Mode == Bundle {
		p.err(p.nr, `catalog link "`+t+`": link another catalog through `+root+major+`/`)
		return
	}
	p.err(p.nr, `catalog link "`+t+`": docs pages link catalogs through `+root+major+`/`)
}

// ownLink checks a link into the bundle's own root: it uses the bundle's
// segment and names a page of the bundle.
func (p *pageLint) ownLink(t, segment, path string) {
	b := p.opts.Bundle
	if segment != b.Segment {
		p.err(p.nr, `link "`+t+`": a link into this bundle uses its own segment, `+b.Root+b.Segment+`/`)
		return
	}
	if path == "" {
		if !p.pages["_index.md"] {
			p.err(p.nr, `link "`+t+`": no page _index.md in this bundle`)
		}
		return
	}
	stem := strings.TrimSuffix(path, "/")
	if !p.pages[stem+".md"] && !p.pages[stem+"/_index.md"] {
		p.err(p.nr, `link "`+t+`": no page `+path+` in this bundle`)
	}
}

func isMajor(s string) bool {
	return s != "edge" && !strings.Contains(s, ".")
}
