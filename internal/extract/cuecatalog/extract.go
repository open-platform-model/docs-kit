package cuecatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// Site links an enforcement row carries.
const decisionsURL = "/enhancements/0010/decisions/"

// fulfilmentProvider is the fulfilment of a contract a platform's catalog
// implements.
const fulfilmentProvider = "provider"

// Options says where the catalog is.
type Options struct {
	Root   string // the source tree: file names and docs/<note>.md are relative to it
	Module string // the module directory, relative to Root ("./opm")
}

// Extract loads the catalog module and returns its doc model.
func Extract(opts Options) (*Model, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, err
	}
	opts.Root = root
	dir := filepath.Join(opts.Root, filepath.FromSlash(opts.Module))
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("cue-catalog module %s: not a directory under %s", opts.Module, opts.Root)
	}
	mod, err := loadModule(root, dir)
	if err != nil {
		return nil, err
	}
	assignPages(mod)
	roots := specRoots(mod)
	model := &Model{
		Schema:       SchemaID,
		ModulePath:   mod.path,
		Version:      mod.version,
		Members:      []Member{},
		Transformers: []Transformer{},
		DocNotes:     []string{},
	}
	notes := map[string]bool{}
	for _, m := range mod.members {
		mm, err := memberModel(mod, m, roots)
		if err != nil {
			return nil, err
		}
		for _, p := range mm.Notes {
			for _, n := range mdtext.DocNotes(p) {
				if !notes[n] && exists(filepath.Join(opts.Root, n)) {
					notes[n] = true
					model.DocNotes = append(model.DocNotes, n)
				}
			}
		}
		model.Members = append(model.Members, mm)
	}
	sort.Strings(model.DocNotes)
	for _, t := range mod.transformers {
		model.Transformers = append(model.Transformers, Transformer{
			Name:              t.name,
			FQN:               t.fqn,
			Description:       t.description,
			RequiredLabels:    labelModels(t.reqLabels),
			RequiredResources: nonNil(t.reqRes),
			OptionalResources: nonNil(t.optRes),
			RequiredTraits:    nonNil(t.reqTraits),
			OptionalTraits:    nonNil(t.optTraits),
		})
	}
	return model, nil
}

func exists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// Encode serializes the model: two-space indent, trailing newline, no HTML
// escaping, so two runs on one source give the same bytes.
func (m *Model) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode reads data/catalog.json.
func Decode(data []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Schema != SchemaID {
		return nil, fmt.Errorf("doc model schema %q, want %q", m.Schema, SchemaID)
	}
	return &m, nil
}

var reAPIVersion = regexp.MustCompile(`^v(\d+)(?:(alpha|beta)(\d+))?$`)

// stability orders apiVersions: the most stable level first, then the
// highest number (v1 > v1beta2 > v1beta1 > v1alpha1).
func stability(v string) [3]int {
	m := reAPIVersion.FindStringSubmatch(v)
	if m == nil {
		return [3]int{-1, 0, 0}
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[3])
	level := 2
	switch m[2] {
	case "alpha":
		level = 0
	case "beta":
		level = 1
	}
	return [3]int{level, major, minor}
}

// NewerAPIVersion reports whether apiVersion a is newer than b in the
// order page paths use: the most stable level first, then the highest
// number.
func NewerAPIVersion(a, b string) bool {
	x, y := stability(a), stability(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return a > b
}

// assignPages names the newest apiVersion of each name and kind
// "<kind>/<name>" and every older one "<kind>/<name>-<apiVersion>".
func assignPages(mod *module) {
	newest := map[string]string{}
	for _, m := range mod.members {
		k := m.kind + "/" + m.name
		if cur, ok := newest[k]; !ok || NewerAPIVersion(m.apiVersion, cur) {
			newest[k] = m.apiVersion
		}
	}
	for _, m := range mod.members {
		m.page = m.kind + "/" + m.name
		if newest[m.kind+"/"+m.name] != m.apiVersion {
			m.page += "-" + m.apiVersion
		}
	}
}

// title is the definition name without its kind suffix: the component
// wrapper a module author embeds (`#ScalingTrait` is `Scaling`).
func title(m *member) string {
	t := strings.TrimPrefix(m.def.name, "#")
	for _, suf := range []string{"Resource", "Trait", "Blueprint"} {
		t = strings.TrimSuffix(t, suf)
	}
	return t
}

var (
	alphaLevel = regexp.MustCompile(`^v\d+alpha\d+$`)
	betaLevel  = regexp.MustCompile(`^v\d+beta\d+$`)
)

func level(apiVersion string) string {
	switch {
	case alphaLevel.MatchString(apiVersion):
		return "alpha"
	case betaLevel.MatchString(apiVersion):
		return "beta"
	default:
		return "GA"
	}
}

func memberModel(mod *module, m *member, roots map[string]*member) (Member, error) {
	paras, err := doctext.SplitDoc(m.doc, m.description)
	if err != nil {
		return Member{}, fmt.Errorf("%s (%s in %s): %w", m.fqn, m.def.name, m.def.filename, err)
	}
	code, linked, external, err := specText(mod, m, roots)
	if err != nil {
		return Member{}, fmt.Errorf("%s: %w", m.fqn, err)
	}
	served := servedBy(mod, m)
	out := Member{
		Kind:              strings.TrimSuffix(m.kind, "s"),
		Name:              m.name,
		APIVersion:        m.apiVersion,
		Level:             level(m.apiVersion),
		FQN:               m.fqn,
		ModulePath:        m.modulePath,
		Title:             title(m),
		Definition:        m.def.name,
		File:              m.def.filename,
		Page:              m.page,
		Description:       m.description,
		Notes:             nonNil(doctext.CleanParagraphs(paras)),
		Category:          optString(m.category),
		Fulfilment:        optString(m.fulfilment),
		Optional:          m.optionalDefault,
		AppliesTo:         nonNil(m.appliesTo),
		ComposedResources: nonNil(m.composedRes),
		ComposedTraits:    nonNil(m.composedTraits),
		MatchLabels:       labelModels(m.matchLabels),
		ServedBy:          served,
		Mark:              mark(m, served),
		Enforcement:       enforcement(m),
		Spec: Spec{
			Key:      m.specKey,
			CUE:      code,
			Fields:   nonNilFields(specFields(m.value.LookupPath(cue.MakePath(cue.Str("spec"), cue.Str(m.specKey))), m.def.pkg.importPath)),
			Linked:   nonNilLinked(linked),
			External: nonNilExternal(external),
		},
	}
	if w, ok := m.def.pkg.defs["#"+title(m)]; ok && w != m.def {
		out.Wrapper = &w.name
	}
	return out, nil
}

// servedBy lists the transformers of the member's own catalog that serve
// it: those that require it, then those that read it when present. A
// blueprint is served by every transformer that requires only what it
// supplies.
func servedBy(mod *module, m *member) []Service {
	out := []Service{}
	for _, t := range mod.transformers {
		var req, opt []string
		switch m.kind {
		case kindResource:
			req, opt = t.reqRes, t.optRes
		case kindTrait:
			req, opt = t.reqTraits, t.optTraits
		case kindBlueprint:
			if blueprintSatisfies(m, t) {
				out = append(out, Service{Transformer: t.name, FQN: t.fqn, Demand: "required"})
			}
			continue
		}
		switch {
		case contains(req, m.fqn):
			out = append(out, Service{Transformer: t.name, FQN: t.fqn, Demand: "required"})
		case contains(opt, m.fqn):
			out = append(out, Service{Transformer: t.name, FQN: t.fqn, Demand: "optional"})
		}
	}
	return out
}

// blueprintSatisfies reports whether a transformer requires only what the
// blueprint supplies: every required label answered by its matchLabels with
// the same value, and every required resource and trait composed by it. A
// transformer that requires nothing at all is never listed.
func blueprintSatisfies(m *member, t *transformer) bool {
	if len(t.reqLabels) == 0 && len(t.reqRes) == 0 && len(t.reqTraits) == 0 {
		return false
	}
	for _, rl := range t.reqLabels {
		ok := false
		for _, ml := range m.matchLabels {
			if ml.key == rl.key && ml.concrete != "" && ml.concrete == rl.concrete {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	for _, r := range t.reqRes {
		if !contains(m.composedRes, r) {
			return false
		}
	}
	for _, r := range t.reqTraits {
		if !contains(m.composedTraits, r) {
			return false
		}
	}
	return true
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// Marks.
const (
	MarkNotImplemented     = "not-implemented"
	MarkProvidedByPlatform = "provided-by-platform"
)

// mark is set for a resource or trait no transformer in its catalog
// serves: provided-by-platform when its fulfilment is provider, else
// not-implemented. A blueprint is never marked.
func mark(m *member, served []Service) *string {
	if len(served) > 0 || m.kind == kindBlueprint {
		return nil
	}
	s := MarkNotImplemented
	if m.fulfilment == fulfilmentProvider {
		s = MarkProvidedByPlatform
	}
	return &s
}

// enforcement lists the rules whose enforcer the source proves: the spec
// schema and each required match label by cue; a provider-fulfilled
// contract's single provider and a load-bearing trait's refused render by
// the kernel.
func enforcement(m *member) []Enforcement {
	noun := strings.TrimSuffix(m.kind, "s")
	code := mdtext.Code
	out := []Enforcement{{
		Rule: fmt.Sprintf("A value under %s satisfies the schema in Spec, or it does not evaluate.", code("spec."+m.specKey)),
		By:   "cue",
	}}
	for _, l := range m.matchLabels {
		if l.required {
			out = append(out, Enforcement{Rule: fmt.Sprintf("A component carrying this %s answers its required match label %s.", noun, code(l.key)), By: "cue"})
		}
	}
	if m.fulfilment == fulfilmentProvider {
		out = append(out, Enforcement{Rule: fmt.Sprintf("A platform carries exactly one catalog whose transformers require this contract; with two, every render on that platform is refused ([0010:D32](%s)).", decisionsURL), By: "kernel"})
	}
	if m.kind == kindTrait && !*m.optionalDefault {
		out = append(out, Enforcement{Rule: fmt.Sprintf("When no transformer matched to the component handles this trait, the render is refused, unless the attachment sets %s ([0010:D28](%s)).", code("optional: true"), decisionsURL), By: "kernel"})
	}
	return out
}

func labelModels(ls []label) []Label {
	out := make([]Label, 0, len(ls))
	for _, l := range ls {
		out = append(out, Label{Key: l.key, Value: l.value, Concrete: l.concrete != "", Required: l.required})
	}
	return out
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

func nonNilFields(xs []Field) []Field {
	if xs == nil {
		return []Field{}
	}
	return xs
}

func nonNilLinked(xs []Linked) []Linked {
	if xs == nil {
		return []Linked{}
	}
	return xs
}

func nonNilExternal(xs []External) []External {
	if xs == nil {
		return []External{}
	}
	return xs
}
