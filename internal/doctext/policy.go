package doctext

import (
	"regexp"
	"strconv"
	"strings"
)

// Policy is what a source's citations become in reader-facing prose.
type Policy string

const (
	// Strip removes every citation (the default).
	Strip Policy = "strip"
	// Link turns each enhancement decision citation into a link to that
	// enhancement's decisions page and removes every other citation form.
	Link Policy = "link"
)

// A decision citation as Link links it: "0010:D28", "0010:D28:R2",
// "0010:D28/D29". The match is refused when it runs on into a form Link
// does not link ("0010:D28/OQ2"), which Strip then removes whole.
var reDecision = regexp.MustCompile(`\b(\d{4}):D\d+(?::R\d+(?:/R\d+)*)?(?:/D\d+(?::R\d+(?:/R\d+)*)?)*`)

// placeholder marks a decision citation while the strip rules run; NUL
// never occurs in source prose and no rule matches it.
func placeholder(i int) string { return "\x00" + strconv.Itoa(i) + "\x00" }

var rePlaceholder = regexp.MustCompile("\x00([0-9]+)\x00")

// Clean cleans one paragraph of prose under the policy. Under Link, a
// decision citation outside a code span becomes
// "[0010:D28](/enhancements/0010/decisions/)", its text as written; a
// citation inside a code span is removed as under Strip.
func (p Policy) Clean(s string) string {
	if p != Link {
		return Clean(s)
	}
	var links []string
	s = outsideCode(s, func(text string) string {
		idx := reDecision.FindAllStringSubmatchIndex(text, -1)
		var b strings.Builder
		last := 0
		for _, m := range idx {
			if m[1] < len(text) && strings.ContainsRune("/:", rune(text[m[1]])) {
				continue
			}
			cite, enh := text[m[0]:m[1]], text[m[2]:m[3]]
			b.WriteString(text[last:m[0]])
			b.WriteString(placeholder(len(links)))
			links = append(links, "["+cite+"](/enhancements/"+enh+"/decisions/)")
			last = m[1]
		}
		b.WriteString(text[last:])
		return b.String()
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
