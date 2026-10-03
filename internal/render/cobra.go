package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/extract/cobra"
	"github.com/open-platform-model/docs-kit/internal/render/helptext"
)

// cobraRenderer renders a cobra data file as a command reference: the
// section index with the global flags, and one page per top-level command
// holding it and every command under it, depth first in name order.
type cobraRenderer struct{}

func (cobraRenderer) Schema() string { return cobra.SchemaID }

func (cobraRenderer) Render(data []byte, t Target) ([]Page, error) {
	m, err := cobra.Decode(data)
	if err != nil {
		return nil, err
	}
	return Commands(m, t)
}

// cobraPage is the frame of a command reference page; Title and
// Description are quoted YAML scalars.
type cobraPage struct {
	Title, Description, Type string
	Weight                   int
	Body                     string
}

// Commands renders the command reference of a cobra doc model.
func Commands(m *cobra.Model, t Target) ([]Page, error) {
	r := &cmdRenderer{m: m, t: t, clean: doctext.Policy(m.Citations).Clean}
	r.section = t.URL(strings.TrimSuffix(m.Section, "/"))
	index, err := execute("cobra.md.tmpl", cobraPage{
		Title: helptext.YAMLQuote(m.Title), Description: helptext.YAMLQuote(m.Description), Weight: m.Weight,
		Body: helptext.EscapeShortcodes(r.index()),
	})
	if err != nil {
		return nil, err
	}
	pages := []Page{{Path: m.Section + "_index.md", Body: index}}
	for i := range m.CLI.Commands {
		top := &m.CLI.Commands[i]
		if top.Page == "" {
			return nil, fmt.Errorf("data/%s: top-level command %q has no page", cobra.DataFile, top.Path)
		}
		body, err := execute("cobra.md.tmpl", cobraPage{
			Title: helptext.YAMLQuote(top.Path), Description: helptext.YAMLQuote(helptext.Sentence(top.Short)), Type: "reference",
			Body: helptext.EscapeShortcodes(r.commandPage(top)),
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{Path: top.Page + ".md", Body: body})
	}
	return pages, nil
}

type cmdRenderer struct {
	m       *cobra.Model
	t       Target
	section string // the section's URL, "/docs/reference/cli/"
	clean   func(string) string
}

// prose formats one line of help text and applies the citation policy.
func (r *cmdRenderer) prose(s string) string { return r.clean(helptext.Prose(s)) }

func (r *cmdRenderer) index() string {
	root := &r.m.CLI
	var b strings.Builder
	if long := helptext.Parse(root.Long); len(long.Desc) > 0 {
		helptext.WriteBlocks(&b, long.Desc, root.Name, r.clean)
		b.WriteString("\n")
	}
	b.WriteString("Each page in this section covers one top-level command and every command under it: " +
		"its usage, description, flags and examples, generated from the CLI's cobra commands. " +
		helptext.CodeSpan(root.Name+" <command> --help") + " prints the same facts for the CLI you have installed.\n\n")
	b.WriteString(helptext.Fence("text", usageLines(root.Path, root.UseLine, root.Runnable, len(root.Commands) > 0)))
	if len(root.GlobalFlags) > 0 {
		b.WriteString("\n## Global flags\n\nEvery command takes these flags.\n\n")
		r.flagTable(&b, root.GlobalFlags)
	}
	return b.String()
}

func (r *cmdRenderer) commandPage(top *cobra.Command) string {
	var b strings.Builder
	if len(r.m.CLI.GlobalFlags) > 0 {
		b.WriteString("Every command on this page also takes the [global flags](" + r.section + "#global-flags).\n")
	}
	walkCommands(top, func(c *cobra.Command) {
		b.WriteString("\n")
		r.entry(&b, c, top.Page)
	})
	return b.String()
}

func walkCommands(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for i := range c.Commands {
		walkCommands(&c.Commands[i], fn)
	}
}

// entry writes one command's entry. Its parts come in a fixed order, each
// only when the command has it: summary, usage (with aliases), description,
// flags, examples, subcommands.
func (r *cmdRenderer) entry(b *strings.Builder, c *cobra.Command, page string) {
	b.WriteString("## " + c.Path + "\n\n")
	if c.Short != "" {
		b.WriteString(helptext.Sentence(r.prose(c.Short)) + "\n\n")
	}
	b.WriteString(helptext.Fence("text", usageLines(c.Path, c.UseLine, c.Runnable, len(c.Commands) > 0)))
	if len(c.Aliases) > 0 {
		spans := make([]string, len(c.Aliases))
		for i, a := range c.Aliases {
			spans[i] = helptext.CodeSpan(a)
		}
		b.WriteString("\nAliases: " + strings.Join(spans, ", ") + ".\n")
	}
	long := helptext.Parse(c.Long)
	if len(long.Desc) > 0 {
		b.WriteString("\n")
		helptext.WriteBlocks(b, long.Desc, r.m.CLI.Name, r.clean)
	}
	flags := append(append([]cobra.Flag{}, c.Flags...), c.InheritedFlags...)
	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	if len(flags) > 0 {
		b.WriteString("\n**Flags**\n\n")
		r.flagTable(b, flags)
	}
	examples := long.Examples
	if c.Example != "" {
		if len(examples) > 0 {
			examples = append(examples, "")
		}
		examples = append(examples, helptext.Dedent(helptext.SplitLines(c.Example))...)
	}
	if len(examples) > 0 {
		b.WriteString("\n**Examples**\n\n" + helptext.Fence("sh", examples))
	}
	if len(c.Commands) > 0 {
		b.WriteString("\n**Subcommands**\n\n| Command | Summary |\n| --- | --- |\n")
		for i := range c.Commands {
			s := &c.Commands[i]
			link := "[" + s.Path + "](" + r.t.URL(page) + "#" + helptext.Anchor(s.Path) + ")"
			b.WriteString("| " + link + " | " + helptext.Cell(helptext.Sentence(r.prose(s.Short))) + " |\n")
		}
	}
}

// usageLines mirrors cobra's usage template: the use line of a runnable
// command, then "<path> [command]" for a command with subcommands.
func usageLines(path, useLine string, runnable, subs bool) []string {
	var lines []string
	if runnable {
		lines = append(lines, useLine)
	}
	if subs {
		lines = append(lines, path+" [command]")
	}
	if len(lines) == 0 {
		lines = append(lines, path)
	}
	return lines
}

func (r *cmdRenderer) flagTable(b *strings.Builder, flags []cobra.Flag) {
	b.WriteString("| Flag | Shorthand | Type | Default | Description |\n| --- | --- | --- | --- | --- |\n")
	for _, f := range flags {
		short := ""
		if f.Shorthand != "" {
			short = helptext.CodeSpan("-" + f.Shorthand)
		}
		def := ""
		if d := flagDefault(f.Default); d != "" {
			def = helptext.CodeSpan(d)
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			helptext.Cell(helptext.CodeSpan("--"+f.Name)), helptext.Cell(short), helptext.Cell(f.Type),
			helptext.Cell(def), helptext.Cell(helptext.Sentence(r.prose(f.Usage))))
	}
}

// flagDefault is the default `--help` would print, or "" where it prints
// none (a zero value).
func flagDefault(d string) string {
	switch d {
	case "", "false", "[]", "0", "0s", "<nil>":
		return ""
	}
	return d
}
