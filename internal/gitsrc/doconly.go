package gitsrc

import (
	"bytes"
	"context"
	"fmt"
	goparser "go/parser"
	goscanner "go/scanner"
	gotoken "go/token"
	"path"
	"regexp"
	"strings"

	cuescanner "cuelang.org/go/cue/scanner"
	cuetoken "cuelang.org/go/cue/token"
)

// Refusal is one change the documentation-only check refuses.
type Refusal struct {
	Path   string
	Reason string
}

func (r Refusal) String() string { return r.Path + ": " + r.Reason }

// patchRelease ends every refusal: the way out of a code change.
const patchRelease = "a change to code needs a patch release"

// DocumentationOnly compares two trees (any tree-ish: a commit, a tag, a
// tree hash) and returns every change that is not documentation. A
// Markdown file may be added, changed, renamed or removed. A .cue or .go
// file may only be changed, and only in its comments: both versions must
// scan to the same token sequence with comments skipped, so a value, a
// string, an attribute or an identifier that changes is refused, while a
// comment added, reworded or removed is not. Layout is not compared, since
// a comment added between two fields moves the code below it. A Go
// directive comment (//go:build, //go:embed, //line, //export) and the cgo
// preamble are code, so a change to either is refused. Everything else is
// refused.
func (r Repo) DocumentationOnly(ctx context.Context, from, to string) ([]Refusal, error) {
	out, err := r.git(ctx, "diff-tree", "-r", "-z", "-M", "--raw", "--no-commit-id", from, to)
	if err != nil {
		return nil, err
	}
	changes, err := parseRaw(out)
	if err != nil {
		return nil, fmt.Errorf("reading the change from %s to %s: %w", from, to, err)
	}
	var refused []Refusal
	for _, c := range changes {
		reason, err := r.judge(ctx, c)
		if err != nil {
			return nil, err
		}
		if reason != "" {
			refused = append(refused, Refusal{Path: c.path, Reason: reason})
		}
	}
	return refused, nil
}

// rawChange is one entry of `git diff-tree --raw`.
type rawChange struct {
	oldMode, newMode string
	oldBlob, newBlob string
	status           byte   // A, C, D, M, R, T, U or X
	path             string // the path after the change
	from             string // the path before a rename or copy
}

// parseRaw reads `git diff-tree -z --raw` output: ":<mode> <mode> <sha>
// <sha> <status>", NUL, the path, and for a rename or copy a second path.
func parseRaw(out string) ([]rawChange, error) {
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	var changes []rawChange
	for i := 0; i < len(fields); i++ {
		if fields[i] == "" {
			continue
		}
		meta := strings.Fields(strings.TrimPrefix(fields[i], ":"))
		if len(meta) != 5 || i+1 >= len(fields) {
			return nil, fmt.Errorf("unexpected entry %q", fields[i])
		}
		c := rawChange{oldMode: meta[0], newMode: meta[1], oldBlob: meta[2], newBlob: meta[3], status: meta[4][0]}
		i++
		c.path = fields[i]
		if c.status == 'R' || c.status == 'C' {
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("rename entry %q has no destination", fields[i])
			}
			c.from = c.path
			i++
			c.path = fields[i]
		}
		changes = append(changes, c)
	}
	return changes, nil
}

func isRegular(mode string) bool { return mode == "100644" || mode == "100755" }

// judge returns why a change is refused, or "".
func (r Repo) judge(ctx context.Context, c rawChange) (string, error) {
	ext := path.Ext(c.path)
	if c.status != 'D' && !isRegular(c.newMode) {
		return "not a regular file (a symlink or a submodule); " + patchRelease, nil
	}
	switch ext {
	case ".md":
		return judgeMarkdown(c), nil
	case ".cue", ".go":
		if c.status != 'M' {
			return fmt.Sprintf("%s; only the comments of an existing %s file may change in a docs revision, and %s", describe(c), ext, patchRelease), nil
		}
		return r.judgeComments(ctx, c, ext)
	default:
		if c.status == 'R' && path.Ext(c.from) != ext {
			return fmt.Sprintf("renamed from %s; only Markdown files, and the comments of .cue and .go files, may change in a docs revision, and %s", c.from, patchRelease), nil
		}
		return fmt.Sprintf("%s, and it is not documentation: only Markdown files, and the comments of .cue and .go files, may change in a docs revision; %s", describe(c), patchRelease), nil
	}
}

func judgeMarkdown(c rawChange) string {
	switch c.status {
	case 'A', 'M', 'D', 'C':
		return ""
	case 'R':
		if path.Ext(c.from) == ".md" {
			return ""
		}
		return fmt.Sprintf("renamed from %s, which is not Markdown; %s", c.from, patchRelease)
	default:
		return fmt.Sprintf("%s; %s", describe(c), patchRelease)
	}
}

func describe(c rawChange) string {
	switch c.status {
	case 'A':
		return "added"
	case 'D':
		return "removed"
	case 'M':
		return "changed"
	case 'R':
		return "renamed from " + c.from
	case 'C':
		return "copied from " + c.from
	case 'T':
		return "its file type changed"
	default:
		return fmt.Sprintf("changed (git status %c)", c.status)
	}
}

func (r Repo) judgeComments(ctx context.Context, c rawChange, ext string) (string, error) {
	before, err := r.blob(ctx, c.oldBlob)
	if err != nil {
		return "", err
	}
	after, err := r.blob(ctx, c.newBlob)
	if err != nil {
		return "", err
	}
	if bytes.Equal(before, after) {
		return "", nil
	}
	if ext == ".cue" {
		return cueCommentsOnly(c.path, before, after), nil
	}
	return goCommentsOnly(c.path, before, after), nil
}

func (r Repo) blob(ctx context.Context, sha string) ([]byte, error) {
	cmd := r.cmd(ctx, "cat-file", "blob", sha)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git cat-file blob %s in %s: %w: %s", sha, r.Dir, err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

// token is one scanned token: its kind and its literal.
type token struct {
	kind string
	lit  string
}

func cueCommentsOnly(name string, before, after []byte) string {
	a, err := cueTokens(name, before)
	if err != nil {
		return fmt.Sprintf("the released version does not scan (%v); %s", err, patchRelease)
	}
	b, err := cueTokens(name, after)
	if err != nil {
		return fmt.Sprintf("the fixed version does not scan (%v); %s", err, patchRelease)
	}
	if !equalTokens(a, b) {
		return "changes CUE values, not only comments; " + patchRelease
	}
	return ""
}

// cueTokens scans CUE source with comments skipped. A comma the scanner
// inserts at a line end and one written in the source are the same token.
// An interpolation is scanned as the parser does: after the parenthesis
// that closes an interpolated expression, the rest of the string is
// resumed.
func cueTokens(name string, src []byte) ([]token, error) {
	var s cuescanner.Scanner
	var errs []string
	f := cuetoken.NewFile(name, 0, len(src))
	s.Init(f, src, func(pos cuetoken.Pos, msg string, args []any) {
		errs = append(errs, fmt.Sprintf("%s: %s", pos, fmt.Sprintf(msg, args...)))
	}, 0)
	var out []token
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
		out = append(out, token{kind: tok.String(), lit: lit})
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
				out = append(out, token{kind: cuetoken.INTERPOLATION.String(), lit: rest})
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

func goCommentsOnly(name string, before, after []byte) string {
	if importsC(name, before) || importsC(name, after) {
		return "imports \"C\", whose preamble comment is code; " + patchRelease
	}
	a, da, err := goTokens(name, before)
	if err != nil {
		return fmt.Sprintf("the released version does not scan (%v); %s", err, patchRelease)
	}
	b, db, err := goTokens(name, after)
	if err != nil {
		return fmt.Sprintf("the fixed version does not scan (%v); %s", err, patchRelease)
	}
	if !equalTokens(a, b) {
		return "changes Go code, not only comments; " + patchRelease
	}
	if !equalTokens(da, db) {
		return "changes a directive comment (such as //go:build or //go:embed), which is code; " + patchRelease
	}
	return ""
}

// reDirective matches the comments the Go toolchain reads as directives:
// "//line ", "//extern ", "//export ", "//<word>:<word>" (//go:build,
// //go:embed, //nolint:...) and the legacy "// +build".
var reDirective = regexp.MustCompile(`^(//(line |extern |export |[a-z0-9]+:[a-z0-9])|/\*line |// \+build)`)

// goTokens scans Go source twice: the tokens with comments skipped, an
// inserted semicolon equal to a written one, and the directive comments
// in order.
func goTokens(name string, src []byte) (code, directives []token, err error) {
	for _, mode := range []goscanner.Mode{0, goscanner.ScanComments} {
		var s goscanner.Scanner
		var errs goscanner.ErrorList
		fset := gotoken.NewFileSet()
		s.Init(fset.AddFile(name, -1, len(src)), src, errs.Add, mode)
		for {
			_, tok, lit := s.Scan()
			if tok == gotoken.EOF {
				break
			}
			switch {
			case mode == 0 && tok == gotoken.SEMICOLON:
				code = append(code, token{kind: tok.String()})
			case mode == 0:
				code = append(code, token{kind: tok.String(), lit: lit})
			case tok == gotoken.COMMENT && reDirective.MatchString(lit):
				directives = append(directives, token{kind: "DIRECTIVE", lit: lit})
			}
		}
		if errs.Len() > 0 {
			return nil, nil, errs.Err()
		}
	}
	return code, directives, nil
}

// importsC reports a Go file that imports "C"; a file that does not parse
// is checked by the scan instead.
func importsC(name string, src []byte) bool {
	f, err := goparser.ParseFile(gotoken.NewFileSet(), name, src, goparser.ImportsOnly)
	if err != nil {
		return false
	}
	for _, imp := range f.Imports {
		if imp.Path != nil && imp.Path.Value == `"C"` {
			return true
		}
	}
	return false
}

func equalTokens(a, b []token) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
