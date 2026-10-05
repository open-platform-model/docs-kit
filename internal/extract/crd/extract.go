package crd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// Options configures one extraction: a crd source of docs-kit.cue.
type Options struct {
	Root    string // the source tree
	Dir     string // "./config/crd/bases", relative to Root
	Samples string // "./config/samples"; "" for no samples
	// HideSamplesMatching: a sample document holding any of these strings
	// is not shown.
	HideSamplesMatching []string
	// StripLabels: each label removed from a shown sample when its value
	// matches.
	StripLabels map[string]string
	// Page is the completable page: Path is the one page's file, or, when
	// Section is set, empty (the section index is derived from Section).
	Page Page
	// Section, "reference/operator/", selects the section layout: an index
	// and one page per kind.
	Section      string
	Order        []string          // kinds first, in this order
	ReconciledBy map[string]string // kind: reconciler name
	Doc          doctext.Policy
	// Outside: the config came from outside the source tree (a backfill),
	// where a missing samples directory is no error.
	Outside bool
}

const crdAPIVersion = "apiextensions.k8s.io/v1"

// Extract reads every CRD under o.Dir and its sample, and returns the doc
// model.
func Extract(o Options) (*Model, error) {
	x := &extraction{o: o}
	kinds, err := x.readKinds()
	if err != nil {
		return nil, err
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("crd dir %s defines no CustomResourceDefinition; point dir at controller-gen's output", o.Dir)
	}
	if err := order(kinds, o); err != nil {
		return nil, err
	}
	citations := CitationsStrip
	if o.Doc == doctext.Link {
		citations = CitationsLink
	}
	m := &Model{Schema: SchemaID, Citations: citations, Layout: LayoutPage, Page: o.Page, Kinds: kinds, Read: x.reads}
	if o.Section != "" {
		if err := m.section(o.Section); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// section lays the model out as a section: the index at
// <section>_index.md, each kind at <section><kind lower-cased>.md.
func (m *Model) section(dir string) error {
	m.Layout = LayoutSection
	m.Page.Path = dir + "_index.md"
	byPage := map[string]string{}
	for i := range m.Kinds {
		k := &m.Kinds[i]
		p := dir + strings.ToLower(k.Kind) + ".md"
		if strings.EqualFold(k.Kind, "index") {
			return fmt.Errorf("the kind %s would be the page %s beside the section index %s_index.md; a section names each kind's page by its lower-cased name, so use the page layout", k.Kind, p, dir)
		}
		if prev, ok := byPage[p]; ok {
			return fmt.Errorf("the kinds %s and %s would both be the page %s; a section names each kind's page by its lower-cased name", prev, k.Kind, p)
		}
		byPage[p] = k.Kind
		k.Page = &p
	}
	return nil
}

// extraction is one Extract run: its options and every file it read.
type extraction struct {
	o     Options
	reads []string
}

// readKinds reads every CRD of o.Dir, with its sample and reconciler.
func (x *extraction) readKinds() ([]Kind, error) {
	o := x.o
	dir, err := within(o.Root, o.Dir)
	if err != nil {
		return nil, err
	}
	if err := checkDir(o.Root, dir, o.Dir, "dir"); err != nil {
		return nil, err
	}
	if err := x.checkSamplesDir(); err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("crd dir %s holds no .yaml file; point dir at controller-gen's output", o.Dir)
	}
	var kinds []Kind
	seen := map[string]string{}
	for _, f := range files {
		rel := relPath(o.Dir, filepath.Base(f))
		docs, err := x.readCRDs(f, rel)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			k, err := extractKind(d, rel, o.Doc)
			if err != nil {
				return nil, err
			}
			if prev, ok := seen[k.Kind]; ok {
				return nil, fmt.Errorf("%s and %s both define the kind %s", prev, rel, k.Kind)
			}
			seen[k.Kind] = rel
			n := len(x.reads)
			if o.Samples != "" {
				if k.Sample, err = x.sample(d); err != nil {
					return nil, err
				}
			}
			if c, ok := o.ReconciledBy[k.Kind]; ok {
				k.ReconciledBy = &c
			}
			k.Read = append([]string{rel}, x.reads[n:]...)
			kinds = append(kinds, k)
		}
	}
	return kinds, nil
}

// order sorts kinds: o.Order's first, in that order, then the rest by
// name. A kind o.Order or o.ReconciledBy names must exist.
func order(kinds []Kind, o Options) error {
	exists := map[string]bool{}
	for i := range kinds {
		exists[kinds[i].Kind] = true
	}
	for _, kind := range sortedKeys(o.ReconciledBy) {
		if !exists[kind] {
			return fmt.Errorf("reconciledBy names the kind %s, which no CRD in %s defines; remove it or fix its name", kind, o.Dir)
		}
	}
	rank := map[string]int{}
	for i, kind := range o.Order {
		if _, dup := rank[kind]; dup {
			return fmt.Errorf("order names the kind %s twice", kind)
		}
		if !exists[kind] {
			return fmt.Errorf("order names the kind %s, which no CRD in %s defines; remove it or fix its name", kind, o.Dir)
		}
		rank[kind] = i
	}
	slices.SortFunc(kinds, func(a, b Kind) int {
		ra, oka := rank[a.Kind]
		rb, okb := rank[b.Kind]
		switch {
		case oka && okb:
			return ra - rb
		case oka:
			return -1
		case okb:
			return 1
		}
		return strings.Compare(a.Kind, b.Kind)
	})
	return nil
}

// within resolves a "./"-relative directory under root and refuses one
// that leaves it.
func within(root, rel string) (string, error) {
	clean := path.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return "", fmt.Errorf("%s leaves the source tree; name a directory inside it", rel)
	}
	return filepath.Join(root, filepath.FromSlash(clean)), nil
}

// relPath is a file's repository-relative slash path: "config/samples/x.yaml".
func relPath(dir, name string) string {
	return path.Join(path.Clean(dir), name)
}

// readCRDs decodes every document of one file, each of which must be a
// CustomResourceDefinition.
func (x *extraction) readCRDs(file, rel string) ([]*crdDoc, error) {
	data, err := x.read(file, rel)
	if err != nil {
		return nil, err
	}
	var out []*crdDoc
	for _, doc := range splitDocuments(data) {
		var head struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
		if err := yaml.Unmarshal(doc, &head); err != nil {
			return nil, fmt.Errorf("%s: does not parse as YAML: %w", rel, err)
		}
		if head.APIVersion == "" && head.Kind == "" {
			continue // an empty document
		}
		if head.APIVersion != crdAPIVersion || head.Kind != "CustomResourceDefinition" {
			return nil, fmt.Errorf("%s: holds a %s %s; a crd dir holds only %s CustomResourceDefinitions", rel, head.APIVersion, head.Kind, crdAPIVersion)
		}
		d, err := decodeCRD(doc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// read reads one regular file of the source tree and records it; a
// symbolic link, a directory or any other file type is refused.
func (x *extraction) read(file, rel string) ([]byte, error) {
	fi, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file (%s); the crd extractor reads only regular files", rel, fi.Mode().Type())
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	x.reads = append(x.reads, rel)
	return data, nil
}

// checkSamplesDir refuses a configured samples directory that is missing
// or not a directory, except in a backfill.
func (x *extraction) checkSamplesDir() error {
	if x.o.Samples == "" {
		return nil
	}
	dir, err := within(x.o.Root, x.o.Samples)
	if err != nil {
		return err
	}
	fi, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist) && x.o.Outside:
		x.o.Samples = ""
		return nil
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("samples %s does not exist; create it or remove samples", x.o.Samples)
	case err != nil:
		return err
	}
	if err := checkMode(fi, x.o.Samples, "samples"); err != nil {
		return err
	}
	return inside(x.o.Root, dir, x.o.Samples, "samples")
}

// checkDir refuses a configured directory that is missing, a symbolic
// link, not a directory, or that resolves outside the source tree.
func checkDir(root, dir, rel, option string) error {
	fi, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s %s does not exist; point %s at controller-gen's output", option, rel, option)
	}
	if err != nil {
		return err
	}
	if err := checkMode(fi, rel, option); err != nil {
		return err
	}
	return inside(root, dir, rel, option)
}

// inside refuses a directory that resolves outside the source tree
// through a linked parent.
func inside(root, dir, rel, option string) error {
	top, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	if r, err := filepath.Rel(top, resolved); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %s resolves outside the source tree; name a directory inside it", option, rel)
	}
	return nil
}

func checkMode(fi fs.FileInfo, rel, option string) error {
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%s %s is a symbolic link; name the directory itself", option, rel)
	case !fi.IsDir():
		return fmt.Errorf("%s %s is not a directory", option, rel)
	}
	return nil
}

// splitDocuments splits a YAML stream at its "---" separator lines (a
// line that starts with "---" followed by nothing but space or a comment),
// as the Kubernetes YAML reader does. Empty documents are dropped.
func splitDocuments(data []byte) [][]byte {
	var docs [][]byte
	var cur bytes.Buffer
	flush := func() {
		if len(bytes.TrimSpace(cur.Bytes())) > 0 {
			docs = append(docs, slices.Clone(cur.Bytes()))
		}
		cur.Reset()
	}
	for _, line := range bytes.SplitAfter(data, []byte("\n")) {
		if rest, ok := bytes.CutPrefix(line, []byte("---")); ok {
			if t := bytes.TrimSpace(rest); len(t) == 0 || t[0] == '#' {
				flush()
				continue
			}
		}
		cur.Write(line)
	}
	flush()
	return docs
}

var (
	reKind      = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	scopes      = []string{"Namespaced", "Cluster"}
	columnTypes = []string{"integer", "number", "string", "boolean", "date"}
	topLevel    = []string{"apiVersion", "kind", "metadata", "spec", "status"}
)

// checkNames refuses names and values the page would write unescaped.
func checkNames(d *crdDoc, rel string) error {
	if !reKind.MatchString(d.Spec.Names.Kind) {
		return fmt.Errorf("%s: kind %q is not a Kubernetes kind name (^[A-Z][A-Za-z0-9]*$)", rel, d.Spec.Names.Kind)
	}
	if !slices.Contains(scopes, d.Spec.Scope) {
		return fmt.Errorf("%s: %s: scope %q is neither Namespaced nor Cluster", rel, d.Spec.Names.Kind, d.Spec.Scope)
	}
	for _, c := range d.Spec.Versions[0].AdditionalPrinterColumns {
		if !slices.Contains(columnTypes, c.Type) {
			return fmt.Errorf("%s: %s: printer column %q has type %q; want one of %s", rel, d.Spec.Names.Kind, c.Name, c.Type, strings.Join(columnTypes, ", "))
		}
	}
	return nil
}

// checkTopLevel refuses top-level properties the page does not show: any
// beside apiVersion, kind, metadata, spec and status, and constraints on
// apiVersion, kind or metadata.
func checkTopLevel(schema props, rel, kind string) error {
	for _, name := range sortedKeys(schema.Properties) {
		if !slices.Contains(topLevel, name) {
			return fmt.Errorf("%s: %s: top-level property %q is not shown; only spec and status are", rel, kind, name)
		}
		if name == "spec" || name == "status" {
			continue
		}
		// apiVersion and kind are plain strings and metadata a plain object;
		// a description and the ignored annotations are allowed, the page shows
		// none of them.
		p := schema.Properties[name]
		p.Description, p.Title, p.Example, p.ExternalDocs, p.XMapType = "", "", nil, nil, nil
		want := `{"type":"string"}`
		if name == "metadata" {
			want = `{"type":"object"}`
		}
		b, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if string(b) != want {
			return fmt.Errorf("%s: %s: %s carries constraints the page does not show: %s", rel, kind, name, b)
		}
	}
	return nil
}

func extractKind(d *crdDoc, rel string, doc doctext.Policy) (Kind, error) {
	if len(d.Spec.Versions) != 1 || d.Spec.Versions[0].Schema == nil || d.Spec.Versions[0].Schema.OpenAPIV3Schema == nil {
		return Kind{}, fmt.Errorf("%s: want exactly one version with a schema; a CRD with several versions is not supported", rel)
	}
	if err := checkNames(d, rel); err != nil {
		return Kind{}, err
	}
	v := d.Spec.Versions[0]
	schema := *v.Schema.OpenAPIV3Schema
	n := d.Spec.Names
	if err := checkTopLevel(schema, rel, n.Kind); err != nil {
		return Kind{}, err
	}
	k := Kind{
		Kind: n.Kind, Group: d.Spec.Group, Plural: n.Plural, Scope: d.Spec.Scope,
		ShortNames: nonNil(n.ShortNames), Categories: nonNil(n.Categories), Subresources: []string{},
		Versions: []Version{{Name: v.Name, Served: v.Served, Storage: v.Storage}},
		File:     rel, Notes: []string{}, Columns: []Column{},
	}
	if s := v.Subresources; s != nil {
		if s.Status != nil {
			k.Subresources = append(k.Subresources, "status")
		}
		if s.Scale != nil {
			k.Subresources = append(k.Subresources, "scale")
		}
	}
	for _, c := range v.AdditionalPrinterColumns {
		k.Columns = append(k.Columns, Column{Name: c.Name, Type: c.Type, JSONPath: c.JSONPath, Priority: c.Priority})
	}
	paras := cleanParagraphs(doc, doctext.Paragraphs(schema.Description))
	if len(paras) > 0 {
		summary, rest := firstSentence(paras[0])
		k.Summary = summary
		if rest != "" {
			k.Notes = append(k.Notes, rest)
		}
		k.Notes = append(k.Notes, paras[1:]...)
	}

	c := &collector{doc: doc, spec: []Field{}, status: []Field{}, rules: []Rule{}}
	c.root(schema)
	// spec and status head their tables rather than being rows, but their
	// own rules (a CEL rule on spec) are recorded like any field's.
	if spec, ok := schema.Properties["spec"]; ok {
		c.constraints("spec", spec)
		c.object("spec", spec, &c.spec)
	}
	if status, ok := schema.Properties["status"]; ok {
		c.constraints("status", status)
		c.object("status", status, &c.status)
	}
	if len(c.unsupported) > 0 {
		slices.Sort(c.unsupported)
		return Kind{}, fmt.Errorf("%s: %s: schema constructs the reference does not render: %s", rel, n.Kind, strings.Join(c.unsupported, "; "))
	}
	k.Spec, k.Status, k.Rules = c.spec, c.status, c.rules
	return k, nil
}

// firstSentence splits a paragraph after its first sentence: the first
// period followed by a space or the end, not ending "e.g." or "i.e.".
func firstSentence(p string) (sentence, rest string) {
	for i := 0; i < len(p); i++ {
		if p[i] != '.' || (i+1 < len(p) && p[i+1] != ' ') {
			continue
		}
		if strings.HasSuffix(p[:i+1], "e.g.") || strings.HasSuffix(p[:i+1], "i.e.") {
			continue
		}
		return p[:i+1], strings.TrimSpace(p[i+1:])
	}
	return p, ""
}

// collector walks a schema into the field tables and the rules.
type collector struct {
	doc          doctext.Policy
	spec, status []Field
	rules        []Rule
	unsupported  []string
}

// formats are the string and integer formats the page shows (as its type,
// or not at all for the integer widths).
var formats = []string{"date-time", "int32", "int64"}

// refuse records every schema construct the reference cannot show, so the
// build fails instead of dropping a rule or a shape silently.
func (c *collector) refuse(at string, s props) {
	if at == "" {
		at = "the object"
	}
	// controller-gen writes an int-or-string field as an anyOf; name it
	// as such rather than as a bare anyOf.
	anyOf := "anyOf"
	if s.XIntOrString {
		anyOf = "an int-or-string field (x-kubernetes-int-or-string), which is not supported yet"
	}
	for _, x := range []struct {
		name    string
		present bool
	}{
		{"allOf", len(s.AllOf) > 0}, {anyOf, len(s.AnyOf) > 0}, {"not", len(s.Not) > 0 && string(s.Not) != "null"},
		{"nullable", s.Nullable}, {"oneOf", len(s.OneOf) > 0},
		{"multipleOf", len(s.MultipleOf) > 0}, {"x-kubernetes-embedded-resource", s.XEmbeddedResource},
		{"tuple items", s.Items != nil && s.Items.Tuple},
		{"format " + strconv.Quote(s.Format), s.Format != "" && !slices.Contains(formats, s.Format)},
		{"messageExpression", slices.ContainsFunc(s.XValidations, func(v validation) bool { return v.MessageExpression != "" })},
	} {
		if x.present {
			c.unsupported = append(c.unsupported, at+": "+x.name)
		}
	}
}

func (c *collector) rule(field, text string, values ...string) {
	c.rules = append(c.rules, Rule{Field: field, Rule: text, Values: nonNil(values), By: EnforcedBy})
}

// root records the rules declared on the object itself: its CEL
// validations and its required top-level fields.
func (c *collector) root(s props) {
	c.refuse("", s)
	c.validations("", s)
	for _, name := range sorted(s.Required) {
		c.rule(name, "Required")
	}
}

// object records a row and the rules for every property of s, recursing
// into nested objects, array items and map values.
func (c *collector) object(prefix string, s props, rows *[]Field) {
	for _, name := range sortedKeys(s.Properties) {
		c.field(prefix+"."+name, s.Properties[name], slices.Contains(s.Required, name), rows)
	}
}

func (c *collector) field(at string, s props, required bool, rows *[]Field) {
	f := Field{Path: at, Type: typeOf(s), Required: required, Description: c.flatten(s.Description), Enum: enumValues(s.Enum)}
	if len(s.Default) > 0 && string(s.Default) != "null" {
		def := string(s.Default)
		f.Default = &def
	}
	*rows = append(*rows, f)
	if required {
		c.rule(at, "Required")
	}
	if isConditions(s) {
		// The standard metav1.Condition list: its own rules are
		// Kubernetes'.
		return
	}
	c.constraints(at, s)
	c.descend(at, s, rows)
}

func (c *collector) descend(at string, s props, rows *[]Field) {
	switch {
	case len(s.Properties) > 0:
		c.object(at, s, rows)
	case s.Items != nil && s.Items.Schema != nil:
		c.constraints(at+"[]", *s.Items.Schema)
		c.descend(at+"[]", *s.Items.Schema, rows)
	case s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil:
		c.constraints(at+".<key>", *s.AdditionalProperties.Schema)
		c.descend(at+".<key>", *s.AdditionalProperties.Schema, rows)
	}
}

// constraints records every rule the API server enforces from the schema
// of one value, apart from required, which the parent declares.
func (c *collector) constraints(at string, s props) {
	c.refuse(at, s)
	c.stringRules(at, s)
	c.numberRules(at, s)
	c.sizeRules(at, s)
	if s.UniqueItems || s.listType("set") {
		c.rule(at, "No duplicate items")
	}
	if s.listType("map") && len(s.XListMapKeys) > 0 {
		c.rule(at, "At most one item per", s.XListMapKeys...)
	}
	c.validations(at, s)
}

func (c *collector) stringRules(at string, s props) {
	if s.MinLength != nil {
		if *s.MinLength == 1 {
			c.rule(at, "Must not be empty")
		} else {
			c.rule(at, fmt.Sprintf("At least %d characters", *s.MinLength))
		}
	}
	if s.MaxLength != nil {
		c.rule(at, fmt.Sprintf("At most %d characters", *s.MaxLength))
	}
	if s.Pattern != "" {
		c.rule(at, "Matches the pattern", s.Pattern)
	}
	if len(s.Enum) > 0 {
		c.rule(at, "One of", enumValues(s.Enum)...)
	}
}

func (c *collector) numberRules(at string, s props) {
	if s.Minimum != nil {
		c.rule(at, bound("At least", "Greater than", *s.Minimum, s.ExclusiveMinimum))
	}
	if s.Maximum != nil {
		c.rule(at, bound("At most", "Less than", *s.Maximum, s.ExclusiveMaximum))
	}
}

func (c *collector) sizeRules(at string, s props) {
	for _, r := range []struct {
		n    *int64
		text string
	}{
		{s.MinItems, "At least %d items"}, {s.MaxItems, "At most %d items"},
		{s.MinProperties, "At least %d entries"}, {s.MaxProperties, "At most %d entries"},
	} {
		if r.n != nil {
			c.rule(at, fmt.Sprintf(r.text, *r.n))
		}
	}
}

func (c *collector) validations(at string, s props) {
	for _, v := range s.XValidations {
		c.rules = append(c.rules, Rule{Field: at, Rule: "CEL rule", Values: []string{v.Rule}, Message: c.flatten(v.Message), By: EnforcedBy})
	}
}

// flatten folds a doc comment onto one line, under the citation policy.
func (c *collector) flatten(s string) string {
	return strings.Join(cleanParagraphs(c.doc, doctext.Paragraphs(s)), " ")
}

// reLinked is a decision citation as the link policy links it.
var reLinked = regexp.MustCompile(`\[(\d{4}:D[0-9DR:/]*)\]\(/enhancements/\d{4}/decisions/\)`)

// cleanParagraphs cleans prose under the policy and keeps it plain: under
// "link" a decision citation stays as written (the renderer links it, from
// the citation alone), every other citation form is removed.
func cleanParagraphs(doc doctext.Policy, paras []string) []string {
	out := doc.CleanParagraphs(paras)
	if doc == doctext.Link {
		for i := range out {
			out[i] = reLinked.ReplaceAllString(out[i], "$1")
		}
	}
	return out
}

const typeArray = "array"

func typeOf(s props) string {
	switch {
	case isConditions(s):
		return "[]Condition"
	case s.XIntOrString:
		return "integer or string"
	case s.Type == typeArray:
		if s.Items != nil && s.Items.Schema != nil {
			return "[]" + typeOf(*s.Items.Schema)
		}
		return typeArray
	case s.Type == "object":
		return objectType(s)
	case s.Type == "string" && s.Format == "date-time":
		return "string (date-time)"
	case s.Type != "":
		return s.Type
	case s.preserveUnknown():
		return "free-form object"
	}
	return "any"
}

func objectType(s props) string {
	switch {
	case s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil:
		return "map[string]" + typeOf(*s.AdditionalProperties.Schema)
	case len(s.Properties) == 0 && s.preserveUnknown():
		return "free-form object"
	}
	return "object"
}

// isConditions reports whether s is the standard metav1.Condition list,
// keyed by type.
func isConditions(s props) bool {
	if s.Type != typeArray || s.Items == nil || s.Items.Schema == nil || !slices.Equal(s.XListMapKeys, []string{"type"}) {
		return false
	}
	item := s.Items.Schema
	for _, p := range []string{"lastTransitionTime", "message", "reason", "status", "type"} {
		if _, ok := item.Properties[p]; !ok || !slices.Contains(item.Required, p) {
			return false
		}
	}
	return true
}

func bound(inclusive, exclusive string, v float64, isExclusive bool) string {
	word := inclusive
	if isExclusive {
		word = exclusive
	}
	return word + " " + strconv.FormatFloat(v, 'f', -1, 64)
}

// enumValues renders each enum value: a string as written, anything else
// as its JSON text.
func enumValues(raw []json.RawMessage) []string {
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		var s string
		if err := json.Unmarshal(r, &s); err == nil {
			out = append(out, s)
			continue
		}
		out = append(out, string(r))
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func sorted(s []string) []string {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
