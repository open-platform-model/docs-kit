// Package placeholder is the version placeholder of a docs page:
// {{opm:<name>.version}}. It owns the list of names, finds the placeholders in
// text, and fills them. The lint refuses an unknown name, and pull fills the
// rest from the versions it collected.
package placeholder

import (
	"fmt"
	"regexp"
	"strings"
)

// Kind says where a name's value comes from.
type Kind int

const (
	// Bundle is the version of a docs bundle of a site version.
	Bundle Kind = iota
	// Unit is an extra version a docs bundle records in its manifest.
	Unit
)

// Name is one entry of the list of names.
type Name struct {
	Name string
	Kind Kind
	// Project is the docs project whose bundle version a Bundle name takes.
	Project string
}

// names is the list, in the order messages print it. A new name is a change
// to this table.
var names = []Name{
	{Name: "cli", Kind: Bundle, Project: "cli"},
	{Name: "core", Kind: Bundle, Project: "core"},
	{Name: "library", Kind: Bundle, Project: "library"},
	{Name: "opm-controller", Kind: Bundle, Project: "opm-controller"},
	{Name: "opm-controller-module", Kind: Unit},
	{Name: "catalog-opm", Kind: Bundle, Project: "catalog-opm-docs"},
}

// Field is the only field a placeholder reads.
const Field = "version"

// Open is the text every placeholder starts with.
const Open = "{{opm:"

// All returns the list of names in list order.
func All() []Name { return append([]Name(nil), names...) }

// Names returns the known names in list order.
func Names() []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n.Name
	}
	return out
}

// Lookup returns the entry of a known name.
func Lookup(name string) (Name, bool) {
	for _, n := range names {
		if n.Name == name {
			return n, true
		}
	}
	return Name{}, false
}

// Verdict is what Scan finds out about one occurrence of the opening text.
type Verdict int

const (
	// Known is a well-formed placeholder with a name on the list.
	Known Verdict = iota
	// UnknownName is well formed, but the name is not on the list.
	UnknownName
	// UnknownField is well formed with a known name, but the field is not "version".
	UnknownField
	// Malformed is not of the form {{opm:<name>.version}}.
	Malformed
)

// Occurrence is one placeholder found in a line.
type Occurrence struct {
	Verdict Verdict
	// Text is the occurrence as written.
	Text string
	// Name and Field are set for every verdict but Malformed.
	Name, Field string
	// Start and End are the byte offsets of Text in the scanned text.
	Start, End int
}

var reWell = regexp.MustCompile(`^\{\{opm:([A-Za-z0-9_-]+)\.([A-Za-z0-9_-]+)\}\}`)

// Scan returns every occurrence of the opening text in s, left to right.
// A malformed occurrence runs to the next blank or the end of s.
func Scan(s string) []Occurrence {
	var out []Occurrence
	for from := 0; ; {
		i := strings.Index(s[from:], Open)
		if i < 0 {
			return out
		}
		start := from + i
		if m := reWell.FindStringSubmatch(s[start:]); m != nil {
			o := Occurrence{Text: m[0], Name: m[1], Field: m[2], Start: start, End: start + len(m[0])}
			switch _, ok := Lookup(o.Name); {
			case !ok:
				o.Verdict = UnknownName
			case o.Field != Field:
				o.Verdict = UnknownField
			}
			out = append(out, o)
			from = o.End
			continue
		}
		end := len(s)
		if j := strings.IndexAny(s[start:], " \t"); j >= 0 {
			end = start + j
		}
		out = append(out, Occurrence{Verdict: Malformed, Text: s[start:end], Start: start, End: end})
		from = end
	}
}

// Message is the lint message of an occurrence that is not Known, or "" for
// a Known one.
func (o Occurrence) Message() string {
	switch o.Verdict {
	case UnknownName:
		return fmt.Sprintf("placeholder %q: unknown name %q; known names: %s", o.Text, o.Name, strings.Join(Names(), ", "))
	case UnknownField:
		return fmt.Sprintf("placeholder %q: unknown field %q; write %s%s.%s}}", o.Text, o.Field, Open, o.Name, Field)
	case Malformed:
		return fmt.Sprintf("placeholder %q: not closed or not of the form %s<name>.%s}}", o.Text, Open, Field)
	case Known:
	}
	return ""
}

// Fill replaces each Known placeholder of s whose name has a value in values.
// Text outside the replaced placeholders is returned unchanged. The second
// result lists every occurrence it left in place: a name without a value, and
// every occurrence that is not Known.
func Fill(s string, values map[string]string) (string, []Occurrence) {
	occ := Scan(s)
	if len(occ) == 0 {
		return s, nil
	}
	var b strings.Builder
	var left []Occurrence
	at := 0
	for _, o := range occ {
		v, ok := values[o.Name]
		if o.Verdict != Known || !ok {
			left = append(left, o)
			continue
		}
		b.WriteString(s[at:o.Start])
		b.WriteString(v)
		at = o.End
	}
	b.WriteString(s[at:])
	return b.String(), left
}
