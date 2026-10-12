package doctext

import (
	"regexp"
	"strconv"
	"strings"
)

// Policy is what a source's citations become in reader-facing prose.
type Policy string

const (
	// Strip removes every citation (the default), and a parenthesis that holds
	// only an enhancement reference.
	Strip Policy = "strip"
	// Link turns each "enhancement NNNN" into a link to that enhancement's
	// page and removes every citation form, as Strip does.
	Link Policy = "link"
)

// An enhancement reference as Link links it: "enhancement 0013", and in the
// plural a list of ids, "enhancements 0013 and 0014" or "enhancements 0013,
// 0014 and 0015". The first letter may be a capital.
var (
	reEnhOne  = regexp.MustCompile(`\b[Ee]nhancement\s+(\d{4})\b`)
	reEnhMany = regexp.MustCompile(`\b[Ee]nhancements\s+(\d{4}(?:(?:,\s*|,?\s+and\s+)\d{4})*)\b`)
	reID      = regexp.MustCompile(`\d{4}`)
	// What follows an id when the reference is really a decision citation
	// (the old form NNNN:Dn or NNNN Dn, or NNNN:OQn) or an experiment
	// pointer, which Clean removes.
	reEnhRefTail = regexp.MustCompile(`^(?::|\s+)(?:D|OQ)\d|^\s+experiment\s+\d`)
	// A parenthesis whose whole content is one enhancement reference or a
	// list of ids, with the space before it.
	reLoneEnh = regexp.MustCompile(`\s*\(\s*(?:[Ee]nhancement\s+\d{4}|[Ee]nhancements\s+\d{4}(?:(?:,\s*|,?\s+and\s+)\d{4})*)\s*\)`)
)

// stripLoneEnhancements removes each parenthesis in s whose whole content
// is an enhancement reference, with the space before it:
// "a rule (enhancement 0013)." becomes "a rule.". A parenthesis that
// follows "]" is a Markdown link destination and stays. The caller passes
// prose outside code spans.
func stripLoneEnhancements(s string) string {
	var b strings.Builder
	last := 0
	for _, m := range reLoneEnh.FindAllStringIndex(s, -1) {
		open := m[0] + strings.IndexByte(s[m[0]:m[1]], '(')
		if open > 0 && s[open-1] == ']' {
			continue
		}
		b.WriteString(s[last:m[0]])
		last = m[1]
	}
	if b.Len() == 0 && last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// LinkEnhancements links each enhancement reference in s that stands alone,
// "enhancement 0013" to [enhancement 0013](/enhancements/0013/) and, in
// "enhancements 0013 and 0014", each id to its page. A reference followed
// by a citation or an experiment pointer is left as written. The caller
// passes prose outside code spans.
func LinkEnhancements(s string) string {
	return linkEnhancements(s, func(md string) string { return md })
}

// linkEnhancements rewrites each linkable reference in s; emit receives the
// Markdown and returns what stands in its place.
func linkEnhancements(s string, emit func(md string) string) string {
	s = rewrite(s, reEnhOne, func(m []int) string {
		id := s[m[2]:m[3]]
		return emit("[" + s[m[0]:m[1]] + "](/enhancements/" + id + "/)")
	})
	return rewrite(s, reEnhMany, func(m []int) string {
		head := s[m[0]:m[2]]
		list := reID.ReplaceAllStringFunc(s[m[2]:m[3]], func(id string) string {
			return emit("[" + id + "](/enhancements/" + id + "/)")
		})
		return head + list
	})
}

// rewrite replaces each match of re in s that no citation tail follows by
// f of its submatch indices.
func rewrite(s string, re *regexp.Regexp, f func(m []int) string) string {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		if reEnhRefTail.MatchString(s[m[1]:]) {
			continue
		}
		b.WriteString(s[last:m[0]])
		b.WriteString(f(m))
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// placeholder marks an enhancement link while the strip rules run; NUL
// never occurs in source prose and no rule matches it.
func placeholder(i int) string { return "\x00" + strconv.Itoa(i) + "\x00" }

var rePlaceholder = regexp.MustCompile("\x00(\\d+)\x00")

// Clean cleans one paragraph of prose under the policy. Under Link,
// "enhancement 0013" outside a code span becomes
// "[enhancement 0013](/enhancements/0013/)", and in "enhancements 0013 and
// 0014" each id becomes a link. Every citation form is removed, as under
// Strip. Under Strip a parenthesis that holds only "enhancement NNNN" is
// removed with the space before it, outside code spans.
func (p Policy) Clean(s string) string {
	if p != Link {
		c := Clean(s)
		if d := outsideCode(c, stripLoneEnhancements); d != c {
			return strings.TrimSpace(d)
		}
		return c
	}
	var links []string
	s = outsideCode(s, func(text string) string {
		return linkEnhancements(text, func(md string) string {
			links = append(links, md)
			return placeholder(len(links) - 1)
		})
	})
	s = Clean(s)
	return rePlaceholder.ReplaceAllStringFunc(s, func(ph string) string {
		i, _ := strconv.Atoi(rePlaceholder.FindStringSubmatch(ph)[1])
		return links[i]
	})
}

// CleanParagraphs is CleanParagraphs under the policy.
func (p Policy) CleanParagraphs(paras []string) []string {
	var out []string
	for _, para := range paras {
		if c := p.Clean(strings.Join(strings.Fields(para), " ")); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// outsideCode applies f to the parts of s outside `code spans`.
func outsideCode(s string, f func(string) string) string {
	parts := strings.Split(s, "`")
	for i := 0; i < len(parts); i += 2 {
		parts[i] = f(parts[i])
	}
	return strings.Join(parts, "`")
}
