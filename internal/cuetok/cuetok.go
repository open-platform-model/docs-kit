// Package cuetok scans CUE source into its token sequence with comments
// skipped, so two texts that differ only in comments and layout compare
// equal. The documentation-only check of a docs revision and the version
// history's spec-text comparison share it.
package cuetok

import (
	"fmt"
	"slices"
	"strings"

	cuescanner "cuelang.org/go/cue/scanner"
	cuetoken "cuelang.org/go/cue/token"
)

// Token is one scanned token: its kind and its literal.
type Token struct {
	Kind string
	Lit  string
}

// Scan scans CUE source with comments skipped; name is used in error
// positions. A comma the scanner inserts at a line end and one written in
// the source are the same token. An interpolation is scanned as the parser
// does: after the parenthesis that closes an interpolated expression, the
// rest of the string is resumed.
func Scan(name string, src []byte) ([]Token, error) {
	var s cuescanner.Scanner
	var errs []string
	f := cuetoken.NewFile(name, 0, len(src))
	s.Init(f, src, func(pos cuetoken.Pos, msg string, args []any) {
		errs = append(errs, fmt.Sprintf("%s: %s", pos, fmt.Sprintf(msg, args...)))
	}, 0)
	var out []Token
	// The parenthesis depth each open interpolation's expression closes
	// at. The scanner returns an interpolation's "(" as its own LPAREN.
	var open []int
	depth := 0
	for len(errs) == 0 {
		_, tok, lit := s.Scan()
		if tok == cuetoken.EOF {
			break
		}
		if tok == cuetoken.COMMA {
			lit = ""
		}
		out = append(out, Token{Kind: tok.String(), Lit: lit})
		switch {
		case tok == cuetoken.INTERPOLATION && strings.HasSuffix(lit, "("):
			open = append(open, depth+1)
		case tok == cuetoken.LPAREN:
			depth++
		case tok == cuetoken.RPAREN:
			depth--
			if n := len(open); n > 0 && open[n-1] == depth+1 {
				open = open[:n-1]
				rest := s.ResumeInterpolation()
				out = append(out, Token{Kind: cuetoken.INTERPOLATION.String(), Lit: rest})
				if strings.HasSuffix(rest, "(") {
					open = append(open, depth+1)
				}
			}
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", errs[0])
	}
	return out, nil
}

// Equal reports two token sequences that are the same.
func Equal(a, b []Token) bool { return slices.Equal(a, b) }
