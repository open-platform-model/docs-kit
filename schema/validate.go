package schema

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	cuejson "cuelang.org/go/encoding/json"
)

// Unify unifies v with the named definition and validates the result as
// concrete. Every error names its file position and field path.
func Unify(def string, v cue.Value) (cue.Value, error) {
	_, d, err := Def(def)
	if err != nil {
		return cue.Value{}, err
	}
	u := d.Unify(v)
	if err := u.Validate(); err != nil {
		return cue.Value{}, Format(err)
	}
	if err := concrete(u); err != nil {
		return cue.Value{}, Format(err)
	}
	return u, nil
}

// concrete checks that every regular field is concrete. It walks the
// fields itself rather than validating the root with cue.Concrete, which
// also reports the open templates of pattern constraints (a #Bundle under
// [#Project]) as unresolved.
func concrete(v cue.Value) error {
	switch v.IncompleteKind() {
	case cue.StructKind:
		it, err := v.Fields()
		if err != nil {
			return err
		}
		for it.Next() {
			if err := concrete(it.Value()); err != nil {
				return err
			}
		}
		return nil
	case cue.ListKind:
		it, err := v.List()
		if err != nil {
			return err
		}
		for it.Next() {
			if err := concrete(it.Value()); err != nil {
				return err
			}
		}
		return nil
	default:
		return v.Validate(cue.Concrete(true))
	}
}

// ValidateJSON validates JSON bytes against the named definition; name is
// the file the bytes came from, used in error positions.
func ValidateJSON(def, name string, data []byte) (cue.Value, error) {
	c, _, err := Def(def)
	if err != nil {
		return cue.Value{}, err
	}
	expr, err := cuejson.Extract(name, data)
	if err != nil {
		return cue.Value{}, fmt.Errorf("%s: %w", name, err)
	}
	v := c.BuildExpr(expr)
	if err := v.Err(); err != nil {
		return cue.Value{}, Format(err)
	}
	return Unify(def, v)
}

// Format turns a CUE error list into one error whose lines read
// "<file>:<line>:<col>: <field path>: <message>", one per distinct error.
func Format(err error) error {
	if err == nil {
		return nil
	}
	seen := map[string]bool{}
	var lines []string
	for _, e := range errors.Errors(err) {
		format, args := e.Msg()
		msg := fmt.Sprintf(format, args...)
		elems := e.Path()
		if len(elems) > 0 && strings.HasPrefix(elems[0], "#") {
			elems = elems[1:] // the definition validated against, not a field of the file
		}
		path := strings.Join(elems, ".")
		where := ""
		for _, p := range e.InputPositions() {
			if p.Filename() != "" && !strings.HasPrefix(p.Filename(), virtualDir) {
				where = p.String()
				break
			}
		}
		if where == "" {
			if p := e.Position(); p.Filename() != "" && !strings.HasPrefix(p.Filename(), virtualDir) {
				where = p.String()
			}
		}
		line := msg
		if path != "" {
			line = path + ": " + line
		}
		if where != "" {
			line = where + ": " + line
		}
		if !seen[line] {
			seen[line] = true
			lines = append(lines, line)
		}
	}
	return &Error{Lines: lines}
}

// Error is a schema validation failure.
type Error struct {
	Lines []string
}

func (e *Error) Error() string {
	return strings.Join(e.Lines, "\n")
}
