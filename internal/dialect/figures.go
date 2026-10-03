package dialect

import "sort"

// Figures returns the figure names a page may embed as {{< opm/<name> >}},
// sorted.
func Figures() []string {
	out := make([]string, 0, len(figures))
	for f := range figures {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}
