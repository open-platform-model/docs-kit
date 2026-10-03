package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/extract/enhancements"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// Enhancements renders the enhancements section's pages: each page's front
// matter, then its body as the enhancements source cleaned it. The bodies
// are the repository's own text, so the source builds them and no
// renderer reads data/enhancements.json for them (docs-kit C21).
func Enhancements(pages []enhancements.Page, t Target) ([]Page, error) {
	if t.Kind != KindSection {
		return nil, fmt.Errorf("the enhancements source writes the /enhancements/ section; give the bundle placement kind \"section\"")
	}
	out := make([]Page, 0, len(pages))
	for _, p := range pages {
		var b strings.Builder
		b.WriteString("---\n")
		b.WriteString("title: " + mdtext.YAMLString(p.Title) + "\n")
		b.WriteString("description: " + mdtext.YAMLString(p.Description) + "\n")
		if p.Type != "" {
			b.WriteString("type: " + p.Type + "\n")
		}
		if p.Weight > 0 {
			b.WriteString("weight: " + strconv.Itoa(p.Weight) + "\n")
		}
		b.WriteString("---\n")
		if body := strings.TrimRight(p.Body, " \t\n"); body != "" {
			b.WriteString("\n" + body + "\n")
		}
		if err := mdtext.CheckShortcodes(p.Path, b.String()); err != nil {
			return nil, err
		}
		out = append(out, Page{Path: p.Path, Body: b.String()})
	}
	return out, nil
}
