package cobradump

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var update = flag.Bool("update", false, "rewrite testdata/dump.golden.json")

const golden = "testdata/dump.golden.json"

func noop(*cobra.Command, []string) {}

// fixtureRoot is a tree with every case the dump decides: hidden,
// deprecated and help-topic commands, the help command, the default
// completion command, root and group persistent flags, hidden and
// deprecated flags, a deprecated shorthand, a home default, aliases and
// Long and Example texts in the shapes the renderer parses.
func fixtureRoot() *cobra.Command {
	root := &cobra.Command{
		Use:     "demo",
		Short:   "Demo manages widgets",
		Long:    "Demo manages widgets and the gadgets they hold.",
		Example: "demo widget list",
		Aliases: []string{"dm"},
	}
	root.Flags().Bool("dry", false, "Dry run")
	root.PersistentFlags().String("config", "/home/u/.demo/config.cue", "Path to config file (env: DEMO_CONFIG)")
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	root.PersistentFlags().Bool("trace", false, "Trace everything")
	_ = root.PersistentFlags().MarkHidden("trace")

	widget := &cobra.Command{
		Use:     "widget",
		Aliases: []string{"w"},
		Short:   "Work with widgets",
		Long: `Work with widgets.

Use this group to create and list widgets. Each widget lives in a 'demo.cue' file
under ./widgets/.`,
	}
	widget.PersistentFlags().StringP("namespace", "n", "default", "Target namespace")
	widget.PersistentFlags().StringP("context", "c", "", "Kubernetes context | cluster")
	_ = widget.PersistentFlags().MarkShorthandDeprecated("context", "use --context")

	create := &cobra.Command{
		Use:   "create <name> [flags]",
		Short: "Create a widget.",
		Long: `Create a widget from a template.

	The template is read from --template <dir>, or the built-in one.

	Arguments:
	  name   The widget name, e.g. web_app
	         (kebab-case).

	Steps:
	  - render the template
	  - write the widget

	Examples:
	  # Create from the built-in template
	  demo widget create web

	  demo widget create web --template ./tpl
`,
		Example: "  demo widget create api\n  demo w create {{< x >}}",
		Run:     noop,
	}
	create.Flags().String("template", "", "Template directory")
	create.Flags().StringSlice("label", nil, "Labels, key=value")
	create.Flags().Duration("timeout", 0, "How long to wait")
	create.Flags().Int("replicas", 1, "Replica count")
	create.Flags().String("old", "", "Old flag")
	_ = create.Flags().MarkDeprecated("old", "use --template")
	create.Flags().String("secret", "", "Hidden flag")
	_ = create.Flags().MarkHidden("secret")
	create.Flags().String("cache", "/home/u/.cache/demo", "Cache directory")
	create.Flags().String("rootfs", "/home/uu/rootfs", "Not under the home directory")

	list := &cobra.Command{Use: "list", Short: "List widgets", Run: noop, Aliases: []string{"ls"}}
	hiddenCmd := &cobra.Command{Use: "secret", Short: "Hidden", Hidden: true, Run: noop}
	oldCmd := &cobra.Command{Use: "old", Short: "Old", Deprecated: "use create", Run: noop}
	widget.AddCommand(list, create, hiddenCmd, oldCmd)

	topic := &cobra.Command{Use: "topic", Short: "A help topic, never listed"}
	version := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Show version information.

Displays:
  - version, commit and build date
  - the SDK version`,
		Run: noop,
	}
	root.AddCommand(widget, version, topic)
	root.InitDefaultHelpCmd()
	return root
}

func dumpFixture(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := Write(fixtureRoot(), &b, Options{Home: "/home/u"}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestWriteGolden(t *testing.T) {
	got := dumpFixture(t)
	if again := dumpFixture(t); !bytes.Equal(got, again) {
		t.Fatal("two dumps of the same tree differ")
	}
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(filepath.Clean(golden))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("dump differs from %s; run go test -run TestWriteGolden -update\n%s", golden, got)
	}
}

func decodeFixture(t *testing.T) dump {
	t.Helper()
	var d dump
	if err := json.Unmarshal(dumpFixture(t), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func names[T any](items []T, name func(T) string) string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, name(it))
	}
	return strings.Join(out, ",")
}

func cmdName(c cmdEntry) string   { return c.Name }
func flagName(f flagEntry) string { return f.Name }

func TestWriteCommands(t *testing.T) {
	d := decodeFixture(t)
	if got := names(d.Commands, cmdName); got != "completion,version,widget" {
		t.Fatalf("top-level commands = %s; want completion listed, help and the help topic left out", got)
	}
	if got := names(d.Commands[2].Commands, cmdName); got != "create,list" {
		t.Fatalf("widget subcommands = %s; want hidden and deprecated left out", got)
	}
	create := d.Commands[2].Commands[0]
	if !strings.HasPrefix(create.Long, "Create a widget from a template.\n\n\tThe template") {
		t.Fatalf("Long is not raw: %q", create.Long)
	}
	if d.Example != "demo widget list" || names(d.Aliases, func(s string) string { return s }) != "dm" || names(d.Flags, flagName) != "dry" {
		t.Fatalf("root example %q, aliases %v, local flags %+v", d.Example, d.Aliases, d.Flags)
	}
}

func TestWriteFlags(t *testing.T) {
	d := decodeFixture(t)
	if got := names(d.GlobalFlags, flagName); got != "config,verbose" {
		t.Fatalf("global flags = %s; want the hidden one left out", got)
	}
	if d.GlobalFlags[0].Default != "~/.demo/config.cue" {
		t.Fatalf("config default = %q; want the home written with ~", d.GlobalFlags[0].Default)
	}
	create := d.Commands[2].Commands[0]
	if got := names(create.Flags, flagName); got != "cache,label,replicas,rootfs,template,timeout" {
		t.Fatalf("create flags = %s", got)
	}
	inh := create.InheritedFlags
	if len(inh) != 2 || inh[0].Name != "context" || inh[0].Shorthand != "" || inh[1].Name != "namespace" || inh[1].Shorthand != "n" {
		t.Fatalf("create inherited flags = %+v; want the group's flags, the deprecated shorthand dropped, no global flag", inh)
	}
}

func TestReplaceHome(t *testing.T) {
	for _, c := range []struct{ home, in, want string }{
		{"/home/u", "/home/u/.demo/config.cue", "~/.demo/config.cue"},
		{"/home/u", "/home/u", "~"},
		{"/home/u/", "/home/u/.cache", "~/.cache"},
		{"/home/u", "[/home/u/a,/home/u/b]", "[~/a,~/b]"},
		{"/root", "/rootfs/x", "/rootfs/x"},
		{"/root", "/root/x", "~/x"},
		{"/", "/etc/demo", "/etc/demo"},
		{"relative", "relative/x", "relative/x"},
	} {
		var b bytes.Buffer
		root := &cobra.Command{Use: "demo"}
		root.PersistentFlags().String("p", c.in, "")
		if err := Write(root, &b, Options{Home: c.home}); err != nil {
			t.Fatal(err)
		}
		var d dump
		if err := json.Unmarshal(b.Bytes(), &d); err != nil {
			t.Fatal(err)
		}
		if got := d.GlobalFlags[0].Default; got != c.want {
			t.Errorf("home %q, default %q: got %q, want %q", c.home, c.in, got, c.want)
		}
	}
}

func TestShorthand(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	root.Flags().StringP("q", "?", "", "")
	if err := Write(root, &bytes.Buffer{}, Options{NoHome: true}); err != nil {
		t.Fatalf("a printable shorthand: %v", err)
	}
}

func TestNoHome(t *testing.T) {
	var b bytes.Buffer
	if err := Write(fixtureRoot(), &b, Options{Home: "/home/u", NoHome: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"/home/u/.demo/config.cue"`) {
		t.Fatal("NoHome rewrote a default")
	}
}

func TestWritePins(t *testing.T) {
	var b bytes.Buffer
	if err := WritePins(&b, map[string]string{"library": "0.5.0", "core": "2.0.0-beta.1"}); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"schema\": \"docs.opmodel.dev/pins/v1\",\n  \"pins\": {\n    \"core\": \"2.0.0-beta.1\",\n    \"library\": \"0.5.0\"\n  }\n}\n"
	if b.String() != want {
		t.Fatalf("pins =\n%s", b.String())
	}
	for _, bad := range []string{"v2.0.0-beta.1", "2.0", "latest"} {
		b.Reset()
		err := WritePins(&b, map[string]string{"core": bad})
		if err == nil || !strings.Contains(err.Error(), "core") || b.Len() != 0 {
			t.Fatalf("%s: err = %v, wrote %q", bad, err, b.String())
		}
	}
}
