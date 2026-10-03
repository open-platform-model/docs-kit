package gitsrc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	goparser "go/parser"
	goscanner "go/scanner"
	gotoken "go/token"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/cuetok"
)

// Refusal is one change the documentation-only check refuses.
type Refusal struct {
	Path   string
	Reason string
}

func (r Refusal) String() string { return r.Path + ": " + r.Reason }

// markdown is the extension of a documentation file.
const markdown = ".md"

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
// preamble are code, so a change to either is refused, and so is moving a
// directive to other code. When a .cue file of the release tree declares
// @extern(embed), CUE may read Markdown as a value, so a Markdown change
// is refused too. A path that is or was a symlink or a submodule is
// refused. Everything else is refused.
func (r Repo) DocumentationOnly(ctx context.Context, from, to string) ([]Refusal, error) {
	out, err := r.git(ctx, "diff-tree", "-r", "-z", "-M", "--raw", "--no-commit-id", from, to)
	if err != nil {
		return nil, err
	}
	changes, err := parseRaw(out)
	if err != nil {
		return nil, fmt.Errorf("reading the change from %s to %s: %w", from, to, err)
	}
	embeds, err := r.embedsFiles(ctx, from)
	if err != nil {
		return nil, err
	}
	var refused []Refusal
	for _, c := range changes {
		reason, err := r.judge(ctx, c)
		if err != nil {
			return nil, err
		}
		if reason == "" && embeds != "" && path.Ext(c.path) == markdown {
			reason = fmt.Sprintf("%s embeds files into CUE values (@extern(embed)), so a Markdown file may be a value; %s", embeds, patchRelease)
		}
		if reason != "" {
			refused = append(refused, Refusal{Path: c.path, Reason: reason})
		}
	}
	return refused, nil
}

// embedsFiles names a .cue file of tree that declares @extern(embed),
// with which CUE reads other files, Markdown included, as values; "" when
// none does. A mention in a comment counts too: the check only refuses.
func (r Repo) embedsFiles(ctx context.Context, tree string) (string, error) {
	cmd := r.cmd(ctx, "grep", "-l", "-F", "-e", "@extern(embed)", tree, "--", "*.cue")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 && errb.Len() == 0 {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("git grep in %s: %w: %s", r.Dir, err, strings.TrimSpace(errb.String()))
	}
	first, _, _ := strings.Cut(strings.TrimSpace(out.String()), "\n")
	_, file, _ := strings.Cut(first, ":")
	return file, nil
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
	if c.status != 'A' && c.status != 'C' && !isRegular(c.oldMode) {
		return "was not a regular file (a symlink or a submodule) in the release; " + patchRelease, nil
	}
	switch ext {
	case markdown:
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
		if path.Ext(c.from) == markdown {
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

// token is one scanned Go token: its kind and its literal.
type token struct {
	kind string
	lit  string
}

func cueCommentsOnly(name string, before, after []byte) string {
	a, err := cuetok.Scan(name, before)
	if err != nil {
		return fmt.Sprintf("the released version does not scan (%v); %s", err, patchRelease)
	}
	b, err := cuetok.Scan(name, after)
	if err != nil {
		return fmt.Sprintf("the fixed version does not scan (%v); %s", err, patchRelease)
	}
	if !cuetok.Equal(a, b) {
		return "changes CUE values, not only comments; " + patchRelease
	}
	return ""
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
	if !slices.Equal(a, b) {
		return "changes Go code, not only comments; " + patchRelease
	}
	if !slices.Equal(da, db) {
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
	var offsets []int // of every code token but a semicolon, in order
	for _, mode := range []goscanner.Mode{0, goscanner.ScanComments} {
		var s goscanner.Scanner
		var errs goscanner.ErrorList
		file := gotoken.NewFileSet().AddFile(name, -1, len(src))
		s.Init(file, src, errs.Add, mode)
		for {
			pos, tok, lit := s.Scan()
			if tok == gotoken.EOF {
				break
			}
			switch {
			case mode == 0 && tok == gotoken.SEMICOLON:
				code = append(code, token{kind: tok.String()})
			case mode == 0:
				code = append(code, token{kind: tok.String(), lit: lit})
				offsets = append(offsets, file.Offset(pos))
			case tok == gotoken.COMMENT && reDirective.MatchString(lit):
				// A directive applies to the code next to it, so it is
				// compared with its place: the number of code tokens
				// before it. Moving //go:embed to another variable is a
				// change.
				at := sort.SearchInts(offsets, file.Offset(pos))
				directives = append(directives, token{kind: "DIRECTIVE@" + strconv.Itoa(at), lit: lit})
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
