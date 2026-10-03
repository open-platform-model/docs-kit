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
	// Home is replaced by "~" wherever it appears in a flag default; ""
	// leaves defaults as they are. Write fills it from os.UserHomeDir when
	// it is unset and NoHome is false.
	Home   string
	NoHome bool
}

type dump struct {
	Schema      string      `json:"schema"`
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Short       string      `json:"short"`
	Long        string      `json:"long"`
	UseLine     string      `json:"useLine"`
	Runnable    bool        `json:"runnable"`
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
// It calls root.InitDefaultCompletionCmd and resolves every command's
// inherited flags before walking, as cobra does on Execute. A hidden or
// deprecated command, the help command, a hidden or deprecated flag and the
// help flag are left out; a deprecated shorthand is dropped. Commands and
// flags are sorted by name, and Long and Example are printed raw.
func Write(root *cobra.Command, w io.Writer, opts Options) error {
	if root == nil {
		return fmt.Errorf("cobradump: no root command")
	}
	if opts.Home == "" && !opts.NoHome {
		if home, err := os.UserHomeDir(); err == nil {
			opts.Home = home
		}
	}
	if opts.NoHome {
		opts.Home = ""
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
		UseLine:     root.UseLine(),
		Runnable:    root.Runnable(),
		GlobalFlags: flags(root.PersistentFlags(), nil, opts),
		Commands:    commands(root, global, opts),
	}
	return encode(w, d)
}

func commands(c *cobra.Command, global func(*pflag.Flag) bool, opts Options) []cmdEntry {
	out := []cmdEntry{}
	for _, s := range available(c) {
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
		def := f.DefValue
		if opts.Home != "" {
			def = strings.ReplaceAll(def, opts.Home, "~")
		}
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
