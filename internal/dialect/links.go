package dialect

import (
	"regexp"
	"strings"
)

// edgeSegment is the segment of a catalog's unreleased build.
const edgeSegment = "edge"

var (
	reExternal    = regexp.MustCompile(`^(https?:|mailto:|#)`)
	reDocs        = regexp.MustCompile(`^/docs/`)
	reDocsOK      = regexp.MustCompile(`^/docs/([a-z0-9-]+/)*(#[^ ]*)?$`)
	reEnh         = regexp.MustCompile(`^/enhancements([/#]|$)`)
	reEnhOK       = regexp.MustCompile(`^/enhancements/(graph/|[0-9][0-9][0-9][0-9]/((problem|design|decisions|graduation|risks|operational|questions)/)?)?(#[^ ]*)?$`)
	reCatalogs    = regexp.MustCompile(`^/catalogs([/#]|$)`)
	reMinor       = regexp.MustCompile(`^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$`)
	reCatalogLink = regexp.MustCompile(`^/catalogs/([a-z0-9]+(?:-[a-z0-9]+)*)/(?:((?:0|[1-9][0-9]*)(?:\.(?:0|[1-9][0-9]*))?|edge)/((?:[a-z0-9-]+/)*))?(#[^ ]*)?$`)
)

// dest checks one link destination.
func (p *pageLint) dest(t string) {
	switch {
	case reExternal.MatchString(t):
	case reDocs.MatchString(t):
		if !reDocsOK.MatchString(t) {
			p.err(p.nr, `internal link "`+t+`": write /docs/<section>/<page>/ with a trailing slash`)
			return
		}
		if p.opts.docsBundle() {
			p.ownedLink(t)
		}
	case reEnh.MatchString(t):
		if !reEnhOK.MatchString(t) {
			p.err(p.nr, `enhancement link "`+t+`": write /enhancements/, /enhancements/<NNNN>/ or /enhancements/<NNNN>/<document>/ with a trailing slash`)
			return
		}
		if p.opts.sectionBundle() {
			p.sectionLink(t)
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
		p.malformedCatalogLink(t)
		return
	}
	name, segment, path := m[1], m[2], m[3]
	root := "/catalogs/" + name + "/"
	if p.opts.Mode == Bundle && !p.opts.docsBundle() && root == p.opts.Bundle.Root {
		p.ownLink(t, segment, path)
		return
	}
	if segment == "" || isMajor(segment) {
		return
	}
	major := "<MAJOR>"
	if segment != edgeSegment {
		major, _, _ = strings.Cut(segment, ".")
	}
	if !p.opts.docsLinkRules() {
		p.err(p.nr, `catalog link "`+t+`": link another catalog through `+root+major+`/`)
		return
	}
	p.err(p.nr, `catalog link "`+t+`": docs pages link catalogs through `+root+major+`/`)
}

// malformedCatalogLink reports a /catalogs/ link that is not of a valid
// form. In docs mode a minor or edge second segment is the fault named,
// even without the trailing slash, so the message matches the site's
// shell lint; anything else is told to take the trailing-slash form.
func (p *pageLint) malformedCatalogLink(t string) {
	if p.opts.docsLinkRules() {
		seg := strings.Split(strings.TrimPrefix(t, "/catalogs/"), "/")
		if len(seg) > 1 {
			switch {
			case reMinor.MatchString(seg[1]):
				major, _, _ := strings.Cut(seg[1], ".")
				p.err(p.nr, `catalog link "`+t+`": docs pages link catalogs through /catalogs/`+seg[0]+`/`+major+`/`)
				return
			case seg[1] == edgeSegment:
				p.err(p.nr, `catalog link "`+t+`": docs pages link catalogs through /catalogs/`+seg[0]+`/<MAJOR>/`)
				return
			}
		}
	}
	p.err(p.nr, `catalog link "`+t+`": write /catalogs/<name>/ or /catalogs/<name>/<MAJOR>/<path>/ with a trailing slash`)
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

// ownedLink checks a /docs/ link of a docs bundle: one into a path the
// bundle owns names a page of the bundle. A link elsewhere points into
// another bundle or a site page, which the site's own link check covers.
func (p *pageLint) ownedLink(t string) {
	path := strings.TrimPrefix(t, "/docs/")
	if i := strings.IndexByte(path, '#'); i >= 0 {
		path = path[:i]
	}
	stem := strings.TrimSuffix(path, "/")
	owned := false
	for _, o := range p.opts.Bundle.Owns {
		if strings.HasSuffix(o, "/") {
			owned = owned || strings.HasPrefix(path, o)
		} else {
			owned = owned || stem+".md" == o
		}
	}
	if !owned {
		return
	}
	if !p.pages[stem+".md"] && !p.pages[stem+"/_index.md"] {
		p.err(p.nr, `link "`+t+`": no page `+path+` in this bundle`)
	}
}

// sectionLink checks a link of a section bundle into its own root: it
// names a page of the bundle. Fragments are not checked.
func (p *pageLint) sectionLink(t string) {
	path := strings.TrimPrefix(t, "/enhancements/")
	if i := strings.IndexByte(path, '#'); i >= 0 {
		path = path[:i]
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
	return s != edgeSegment && !strings.Contains(s, ".")
}
