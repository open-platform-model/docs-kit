// Package cobradump prints a cobra command tree, and a program's pins, as
// the JSON documents opm-docs reads (docs-kit C19). It depends on cobra and
// pflag only, so a CLI that imports it gains nothing else in its module
// graph.
//
// A CLI calls it from a small program of its own, never from the binary it
// ships:
//
//	func main() {
//		if len(os.Args) > 1 && os.Args[1] == "pins" {
//			check(cobradump.WritePins(os.Stdout, map[string]string{"core": "2.0.0"}))
//			return
//		}
//		check(cobradump.Write(cmd.NewRootCmd(), os.Stdout, cobradump.Options{}))
//	}
package cobradump

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Schema ids of the two documents.
const (
	Schema     = "docs.opmodel.dev/cobradump/v1"
	PinsSchema = "docs.opmodel.dev/pins/v1"
)

// Options adjusts a dump.
type Options struct {
	// Home is replaced by "~" in a flag default wherever it is followed by
	// a path separator or ends the default, and only when it is an absolute
	// path other than the file-system root; "" leaves defaults as they are.
	// Write fills it from os.UserHomeDir when it is unset and NoHome is
	// false.
	Home   string
	NoHome bool
}

type dump struct {
	Schema      string      `json:"schema"`
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Short       string      `json:"short"`
	Long        string      `json:"long"`
	Example     string      `json:"example"`
	Aliases     []string    `json:"aliases"`
	UseLine     string      `json:"useLine"`
	Runnable    bool        `json:"runnable"`
	Flags       []flagEntry `json:"flags"`
	GlobalFlags []flagEntry `json:"globalFlags"`
	Commands    []cmdEntry  `json:"commands"`
}

type cmdEntry struct {
	Path           string      `json:"path"`
	Name           string      `json:"name"`
	Use            string      `json:"use"`
	UseLine        string      `json:"useLine"`
	Short          string      `json:"short"`
	Long           string      `json:"long"`
	Example        string      `json:"example"`
	Aliases        []string    `json:"aliases"`
	Runnable       bool        `json:"runnable"`
	Flags          []flagEntry `json:"flags"`
	InheritedFlags []flagEntry `json:"inheritedFlags"`
	Commands       []cmdEntry  `json:"commands"`
}

type flagEntry struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand"`
	Type      string `json:"type"`
	Default   string `json:"default"`
	Usage     string `json:"usage"`
}

// Write prints root's tree as one docs.opmodel.dev/cobradump/v1 document.
// The format is closed: any change to its fields is a new schema,
// docs.opmodel.dev/cobradump/v2.
// It calls root.InitDefaultCompletionCmd and resolves every command's
// inherited flags before walking, as cobra does on Execute. A hidden or
// deprecated command, the help command, a hidden or deprecated flag and the
// help flag are left out; a deprecated shorthand is dropped. Commands and
// flags are sorted by name, and Long and Example are printed raw. A flag
// shorthand that is not one printable ASCII character is an error.
func Write(root *cobra.Command, w io.Writer, opts Options) error {
	if root == nil {
		return fmt.Errorf("cobradump: no root command")
	}
	if opts.Home == "" && !opts.NoHome {
		if home, err := os.UserHomeDir(); err == nil {
			opts.Home = home
		}
	}
	if opts.NoHome || !usableHome(opts.Home) {
		opts.Home = ""
	} else {
		opts.Home = filepath.Clean(opts.Home)
	}
	root.InitDefaultCompletionCmd()
	// cobra merges a parent's persistent flags into a command only when it
	// runs; merge them everywhere first, so use lines and inherited flags
	// read as `--help` prints them.
	walk(root, func(c *cobra.Command) { c.InheritedFlags() })

	global := func(f *pflag.Flag) bool { return root.PersistentFlags().Lookup(f.Name) != nil }
	d := dump{
		Schema:      Schema,
		Name:        root.Name(),
		Path:        root.CommandPath(),
		Short:       root.Short,
		Long:        root.Long,
		Example:     root.Example,
		Aliases:     append([]string{}, root.Aliases...),
		UseLine:     root.UseLine(),
		Runnable:    root.Runnable(),
		Flags:       flags(root.LocalNonPersistentFlags(), nil, opts),
		GlobalFlags: flags(root.PersistentFlags(), nil, opts),
		Commands:    commands(root, global, opts),
	}
	if err := checkShorthands(d.Path, d.Flags, d.GlobalFlags, d.Commands); err != nil {
		return err
	}
	return encode(w, d)
}

// checkShorthands refuses a shorthand that is not one printable ASCII
// character, which opm-docs would refuse too.
func checkShorthands(path string, local, other []flagEntry, cmds []cmdEntry) error {
	for _, f := range append(append([]flagEntry{}, local...), other...) {
		if !reShorthand.MatchString(f.Shorthand) {
			return fmt.Errorf("cobradump: %s: flag --%s has the shorthand %q, not one printable ASCII character", path, f.Name, f.Shorthand)
		}
	}
	for i := range cmds {
		c := &cmds[i]
		if err := checkShorthands(c.Path, c.Flags, c.InheritedFlags, c.Commands); err != nil {
			return err
		}
	}
	return nil
}

var reShorthand = regexp.MustCompile(`^[\x21-\x7E]?$`)

// usableHome reports a home directory worth replacing: absolute and not
// the file-system root.
func usableHome(home string) bool {
	return filepath.IsAbs(home) && strings.TrimRight(home, `/\`) != "" && filepath.Dir(home) != home
}

// replaceHome writes home as "~" in s wherever it is followed by a path
// separator or ends s, so /root is replaced in /root/x but not in /rootfs.
func replaceHome(s, home string) string {
	if home == "" {
		return s
	}
	var b strings.Builder
	for {
		i := strings.Index(s, home)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := i + len(home)
		if end == len(s) || s[end] == '/' || s[end] == filepath.Separator {
			b.WriteString(s[:i] + "~")
		} else {
			b.WriteString(s[:end])
		}
		s = s[end:]
	}
}

func commands(c *cobra.Command, global func(*pflag.Flag) bool, opts Options) []cmdEntry {
	subs := available(c)
	out := make([]cmdEntry, 0, len(subs))
	for _, s := range subs {
		aliases := append([]string{}, s.Aliases...)
		out = append(out, cmdEntry{
			Path:           s.CommandPath(),
			Name:           s.Name(),
			Use:            s.Use,
			UseLine:        s.UseLine(),
			Short:          s.Short,
			Long:           s.Long,
			Example:        s.Example,
			Aliases:        aliases,
			Runnable:       s.Runnable(),
			Flags:          flags(s.LocalFlags(), global, opts),
			InheritedFlags: flags(s.InheritedFlags(), global, opts),
			Commands:       commands(s, global, opts),
		})
	}
	return out
}

// available returns the subcommands `--help` lists, sorted by name.
func available(c *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, s := range c.Commands() {
		if s.IsAvailableCommand() && s.Name() != "help" {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, s := range available(c) {
		walk(s, fn)
	}
}

// flags lists fs's visible flags, sorted by name, leaving out those skip
// reports.
func flags(fs *pflag.FlagSet, skip func(*pflag.Flag) bool, opts Options) []flagEntry {
	out := []flagEntry{}
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" || f.Name == "help" || (skip != nil && skip(f)) {
			return
		}
		short := f.Shorthand
		if f.ShorthandDeprecated != "" {
			short = ""
		}
		def := replaceHome(f.DefValue, opts.Home)
		out = append(out, flagEntry{Name: f.Name, Shorthand: short, Type: f.Value.Type(), Default: def, Usage: f.Usage})
	})
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// reSemVer is an exact SemVer version without "v".
var reSemVer = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// WritePins prints one docs.opmodel.dev/pins/v1 document: project name to
// exact version, without "v". A version that is not one writes nothing and
// returns an error naming its project.
func WritePins(w io.Writer, pins map[string]string) error {
	names := make([]string, 0, len(pins))
	for p := range pins {
		names = append(names, p)
	}
	sort.Strings(names)
	for _, p := range names {
		if v := pins[p]; !reSemVer.MatchString(v) {
			return fmt.Errorf("cobradump: pin %s is %q, not an exact version such as 1.0.0-beta.1 (no v)", p, v)
		}
	}
	if pins == nil {
		pins = map[string]string{}
	}
	return encode(w, struct {
		Schema string            `json:"schema"`
		Pins   map[string]string `json:"pins"`
	}{PinsSchema, pins})
}

// encode writes v as indented JSON with one trailing newline, in one
// write, so a failed encoding prints nothing.
func encode(w io.Writer, v any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("cobradump: %w", err)
	}
	_, err := w.Write(b.Bytes())
	return err
}
