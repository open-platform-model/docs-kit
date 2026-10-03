package dialect

import (
	"regexp"
	"strings"
)

// The patterns below are the shell lint's awk patterns. The ones used to
// find a match inside a line run leftmost-longest, as POSIX awk's match does.
var (
	reFMKey     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*:`)
	reFMValue   = regexp.MustCompile(`^[^:]*:[ \t]*`)
	reFMClose   = regexp.MustCompile(`^---[ \t]*$`)
	reAllowed   = regexp.MustCompile(`^(title|description|type|weight)$`)
	reShortcode = longest(`\{\{[<%][ \t]*/?[ \t]*[^ \t>%}]*`)
	reShown     = regexp.MustCompile(`^\{\{[<%][ \t]*/\*`)
	reSCPrefix  = longest(`^\{\{[<%][ \t]*/?[ \t]*`)
	reSCOpen    = regexp.MustCompile(`^\{\{<[ \t]*opm`)
	reSCClose   = regexp.MustCompile(`^[ \t]*>\}\}`)
	reAside     = regexp.MustCompile(`^[ \t]*:::`)
	reImport    = regexp.MustCompile(`^[ \t]*import[ \t].*[ \t]from[ \t]`)
	reFence     = regexp.MustCompile("^ ? ? ?(```|~~~)")
	reFenceTag  = longest("^[`~]*[ \t]*")
	reComponent = regexp.MustCompile(`^[ \t]*<[A-Z][A-Za-z0-9]*([ \t][^>]*)?/?>[ \t]*$`)
	reAlert     = regexp.MustCompile(`^[ \t]*>[ \t]*\[!`)
	reAlertOK   = regexp.MustCompile(`^> \[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$`)
	reImage     = regexp.MustCompile(`!\[|<[Ii][Mm][Gg][ \t>/]`)
	reRawHTML   = regexp.MustCompile(`[Hh][Rr][Ee][Ff][ \t]*=|[Ss][Rr][Cc][ \t]*=`)
	reRefDef    = regexp.MustCompile(`^ ? ? ?\[[^\]^][^\]]*\]:`)
	reRefPrefix = longest(`^ ? ? ?\[[^\]]+\]:[ \t]*`)
	reRefTail   = longest(`[ \t].*$`)
	reInline    = longest(`\]\([^) \t]*`)
	reComment   = longest(`[ \t]+#.*$`)
	reTrailWS   = longest(`[ \t]+$`)
	reType      = regexp.MustCompile(`^(tutorial|how-to|explanation|reference)$`)
	reWeight    = regexp.MustCompile(`^[1-9]\d*$`)
)

func longest(expr string) *regexp.Regexp {
	re := regexp.MustCompile(expr)
	re.Longest()
	return re
}

// pageLint is the per-file state of the shell lint's awk program.
type pageLint struct {
	file  string
	leaf  bool
	opts  Options
	pages map[string]bool

	nr     int // current line number
	nofm   bool
	infm   bool
	fmdone bool
	fence  string
	line   map[string]int
	val    map[string]string
	out    []Violation
}

func (p *pageLint) err(line int, msg string) {
	p.out = append(p.out, Violation{File: p.file, Line: line, Msg: msg})
}

func (p *pageLint) run(lines []string) {
	for i, l := range lines {
		p.nr = i + 1
		p.record(l)
	}
	p.end()
}

func (p *pageLint) record(l string) {
	if p.nr == 1 {
		if l != "---" {
			p.err(1, "front matter must open on line 1 with ---")
			p.nofm = true
		} else {
			p.infm = true
			return
		}
	}
	if p.infm {
		p.frontMatter(l)
		return
	}
	p.shortcodes(l)
	if reAside.MatchString(l) {
		p.err(p.nr, "Starlight aside (:::); write a GitHub alert: > [!NOTE]")
	}
	if reImport.MatchString(l) {
		p.err(p.nr, "MDX import line")
	}
	if p.fenceLine(l) || p.fence != "" {
		return
	}
	p.body(l)
}

func (p *pageLint) frontMatter(l string) {
	if reFMClose.MatchString(l) {
		p.infm = false
		p.fmdone = true
		return
	}
	if reFMKey.MatchString(l) {
		k, _, _ := strings.Cut(l, ":")
		v := reFMValue.ReplaceAllString(l, "")
		p.line[k] = p.nr
		p.val[k] = v
		if k == "sidebar" {
			p.err(p.nr, "sidebar: is Starlight front matter; write weight: N")
		} else if !reAllowed.MatchString(k) {
			p.err(p.nr, "front-matter key \""+k+"\" is not allowed (title, description, type, weight)")
		}
		return
	}
	if !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") && !strings.HasPrefix(l, "#") && l != "" {
		p.err(p.nr, `front matter: expected "key: value"`)
	}
}

// shortcodes checks every shortcode on a line. Hugo expands shortcodes even
// inside code fences, so this runs on every line.
func (p *pageLint) shortcodes(l string) {
	s := l
	for {
		loc := reShortcode.FindStringIndex(s)
		if loc == nil {
			return
		}
		tok, rest := s[loc[0]:loc[1]], s[loc[1]:]
		s = rest
		if reShown.MatchString(tok) {
			continue
		}
		name := reSCPrefix.ReplaceAllString(tok, "")
		if !strings.HasPrefix(name, "opm/") {
			p.err(p.nr, "shortcode \""+name+"\": source pages use only {{< opm/<figure> >}}")
			continue
		}
		if !figures[name[4:]] {
			p.err(p.nr, "unknown figure shortcode \""+name+"\"")
			continue
		}
		if !reSCOpen.MatchString(tok) || !reSCClose.MatchString(rest) {
			p.err(p.nr, "write the figure as {{< "+name+" >}} (no parameters, no closing tag)")
		}
	}
}

// fenceLine handles a fence opening or closing line, reporting whether the
// line was consumed. A fence closes only on a run of its own character at
// least as long as the run that opened it, with nothing after it but
// spaces; any other fence line inside a fence is not consumed, and the
// caller skips it as fenced content.
func (p *pageLint) fenceLine(l string) bool {
	if !reFence.MatchString(l) {
		return false
	}
	m := strings.TrimLeft(l, " ")
	run := m[:len(m)-len(strings.TrimLeft(m, m[:1]))]
	if p.fence == "" {
		p.fence = run
		if reFenceTag.ReplaceAllString(m, "") == "" {
			p.err(p.nr, "code fence without a language tag; write ```text for plain text")
		}
		return true
	}
	if run[0] == p.fence[0] && len(run) >= len(p.fence) && strings.TrimSpace(m[len(run):]) == "" {
		p.fence = ""
		return true
	}
	return false
}

func (p *pageLint) body(l string) {
	if reComponent.MatchString(l) {
		p.err(p.nr, "component tag; use {{< opm/<figure> >}}")
	}
	if reAlert.MatchString(l) && !reAlertOK.MatchString(l) {
		p.err(p.nr, `alert marker must be exactly "> [!NOTE]" (or TIP, IMPORTANT, WARNING, CAUTION) alone on its line`)
	}
	if reImage.MatchString(l) {
		p.err(p.nr, "image; docs/site pages carry no images")
	}
	if reRawHTML.MatchString(l) {
		p.err(p.nr, "raw HTML link or source; write a Markdown link [text](/docs/<section>/<page>/)")
	}
	if reRefDef.MatchString(l) {
		t := reRefPrefix.ReplaceAllString(l, "")
		t = reRefTail.ReplaceAllString(t, "")
		t = strings.TrimPrefix(t, "<")
		t = strings.TrimSuffix(t, ">")
		if t != "" {
			p.dest(t)
		}
	}
	s := l
	for {
		loc := reInline.FindStringIndex(s)
		if loc == nil {
			break
		}
		t := s[loc[0]+2 : loc[1]]
		s = s[loc[1]:]
		p.dest(t)
	}
}

func (p *pageLint) end() {
	if p.nofm {
		return
	}
	if !p.fmdone {
		p.err(1, "front matter is not closed with ---")
		return
	}
	if v, ok := p.val["title"]; !ok || unq(v) == "" {
		p.err(1, "missing title")
	}
	if v, ok := p.val["description"]; !ok || unq(v) == "" {
		p.err(1, "missing description")
	}
	t, hasType := p.val["type"]
	switch {
	case p.leaf && !hasType:
		p.err(1, "missing type (tutorial, how-to, explanation or reference)")
	case p.leaf:
		if u := unq(t); !reType.MatchString(u) {
			p.err(p.line["type"], "invalid type \""+u+"\"")
		}
	case hasType:
		p.err(p.line["type"], "a section overview (_index.md) declares no type")
	}
	if w, ok := p.val["weight"]; ok && !reWeight.MatchString(unq(w)) {
		p.err(p.line["weight"], "weight must be a positive integer")
	}
}

// unq strips a trailing comment, trailing blanks and one pair of matching
// quotes from a front-matter value.
func unq(s string) string {
	s = reComment.ReplaceAllString(s, "")
	s = reTrailWS.ReplaceAllString(s, "")
	if len(s) >= 2 && s[0] == s[len(s)-1] && (s[0] == '"' || s[0] == '\'') {
		s = s[1 : len(s)-1]
	}
	return s
}
