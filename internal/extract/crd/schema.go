package crd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"sigs.k8s.io/yaml"
)

// The fields of an apiextensions.k8s.io/v1 CustomResourceDefinition the
// extractor reads. A local struct rather than the Kubernetes types, which
// would add the Kubernetes API machinery to the module graph for a handful
// of fields. Decoding is strict: a field the struct does not name is
// refused, so nothing the page cannot show is dropped silently.

type crdDoc struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name        string            `json:"name"`
		Annotations map[string]string `json:"annotations"`
		Labels      map[string]string `json:"labels"`
		// Older controller-gen writes "creationTimestamp: null".
		CreationTimestamp *string `json:"creationTimestamp"`
	} `json:"metadata"`
	Spec crdSpec `json:"spec"`
	// The status block a CRD read back from a cluster carries; ignored.
	Status json.RawMessage `json:"status"`
}

type crdSpec struct {
	Group    string       `json:"group"`
	Scope    string       `json:"scope"`
	Names    crdNames     `json:"names"`
	Versions []crdVersion `json:"versions"`
	// Only false is accepted: true is the v1beta1 pruning opt-out, which an
	// apiextensions.k8s.io/v1 CRD cannot set.
	PreserveUnknownFields *bool `json:"preserveUnknownFields"`
}

type crdNames struct {
	Kind       string   `json:"kind"`
	ListKind   string   `json:"listKind"`
	Plural     string   `json:"plural"`
	Singular   string   `json:"singular"`
	ShortNames []string `json:"shortNames"`
	Categories []string `json:"categories"`
}

type crdVersion struct {
	Name    string `json:"name"`
	Served  bool   `json:"served"`
	Storage bool   `json:"storage"`
	// Deprecation enforces nothing; ignored.
	Deprecated         bool   `json:"deprecated"`
	DeprecationWarning string `json:"deprecationWarning"`
	Schema             *struct {
		OpenAPIV3Schema *props `json:"openAPIV3Schema"`
	} `json:"schema"`
	Subresources *struct {
		Status *json.RawMessage `json:"status"`
		Scale  *json.RawMessage `json:"scale"`
	} `json:"subresources"`
	AdditionalPrinterColumns []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Format      string `json:"format"`
		Description string `json:"description"`
		JSONPath    string `json:"jsonPath"`
		Priority    int    `json:"priority"`
	} `json:"additionalPrinterColumns"`
}

// props is one OpenAPI v3 schema node. Every field is omitempty, so a node
// re-encodes to exactly the keys it was written with.
type props struct {
	Type                 string            `json:"type,omitempty"`
	Format               string            `json:"format,omitempty"`
	Description          string            `json:"description,omitempty"`
	Properties           map[string]props  `json:"properties,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Items                *orArray          `json:"items,omitempty"`
	AdditionalProperties *orBool           `json:"additionalProperties,omitempty"`
	Default              json.RawMessage   `json:"default,omitempty"`
	Enum                 []json.RawMessage `json:"enum,omitempty"`
	Pattern              string            `json:"pattern,omitempty"`
	MinLength            *int64            `json:"minLength,omitempty"`
	MaxLength            *int64            `json:"maxLength,omitempty"`
	MinItems             *int64            `json:"minItems,omitempty"`
	MaxItems             *int64            `json:"maxItems,omitempty"`
	MinProperties        *int64            `json:"minProperties,omitempty"`
	MaxProperties        *int64            `json:"maxProperties,omitempty"`
	Minimum              *float64          `json:"minimum,omitempty"`
	Maximum              *float64          `json:"maximum,omitempty"`
	ExclusiveMinimum     bool              `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     bool              `json:"exclusiveMaximum,omitempty"`
	UniqueItems          bool              `json:"uniqueItems,omitempty"`
	XIntOrString         bool              `json:"x-kubernetes-int-or-string,omitempty"`
	XPreserveUnknown     *bool             `json:"x-kubernetes-preserve-unknown-fields,omitempty"`
	XListType            *string           `json:"x-kubernetes-list-type,omitempty"`
	XListMapKeys         []string          `json:"x-kubernetes-list-map-keys,omitempty"`
	XValidations         []validation      `json:"x-kubernetes-validations,omitempty"`

	// Annotations that enforce no rule the page would miss; ignored.
	XMapType     *string         `json:"x-kubernetes-map-type,omitempty"`
	Title        string          `json:"title,omitempty"`
	Example      json.RawMessage `json:"example,omitempty"`
	ExternalDocs json.RawMessage `json:"externalDocs,omitempty"`

	// Constructs the reference cannot show: decoded only to be refused
	// with a message naming them.
	Nullable          bool              `json:"nullable,omitempty"`
	AllOf             []json.RawMessage `json:"allOf,omitempty"`
	OneOf             []json.RawMessage `json:"oneOf,omitempty"`
	AnyOf             []json.RawMessage `json:"anyOf,omitempty"`
	Not               json.RawMessage   `json:"not,omitempty"`
	MultipleOf        json.RawMessage   `json:"multipleOf,omitempty"`
	XEmbeddedResource bool              `json:"x-kubernetes-embedded-resource,omitempty"`
}

type validation struct {
	Rule              string `json:"rule"`
	Message           string `json:"message,omitempty"`
	MessageExpression string `json:"messageExpression,omitempty"` // refused
	Reason            string `json:"reason,omitempty"`            // ignored: the status reason of a refusal
	FieldPath         string `json:"fieldPath,omitempty"`         // ignored: where the refusal points
}

// orArray is items: one schema, or a list of them (a tuple), which is
// refused.
type orArray struct {
	Schema *props
	Tuple  bool
}

func (o *orArray) UnmarshalJSON(b []byte) error {
	if bytes.HasPrefix(bytes.TrimSpace(b), []byte("[")) {
		o.Tuple = true
		return nil
	}
	o.Schema = &props{}
	return strictJSON(b, o.Schema)
}

func (o orArray) MarshalJSON() ([]byte, error) {
	if o.Tuple {
		return []byte("[]"), nil
	}
	return json.Marshal(o.Schema)
}

// orBool is additionalProperties: a schema, or true or false.
type orBool struct {
	Schema *props
	Bool   *bool
}

func (o *orBool) UnmarshalJSON(b []byte) error {
	t := bytes.TrimSpace(b)
	if bytes.Equal(t, []byte("true")) || bytes.Equal(t, []byte("false")) {
		v := bytes.Equal(t, []byte("true"))
		o.Bool = &v
		return nil
	}
	o.Schema = &props{}
	return strictJSON(b, o.Schema)
}

func (o orBool) MarshalJSON() ([]byte, error) {
	if o.Bool != nil {
		return json.Marshal(*o.Bool)
	}
	return json.Marshal(o.Schema)
}

func (p props) preserveUnknown() bool { return p.XPreserveUnknown != nil && *p.XPreserveUnknown }

func (p props) listType(t string) bool { return p.XListType != nil && *p.XListType == t }

// strictJSON decodes b into v, refusing a field v does not name.
func strictJSON(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// decodeCRD decodes one YAML document strictly: a key the struct does not
// name is refused with its path.
func decodeCRD(doc []byte) (*crdDoc, error) {
	j, err := yaml.YAMLToJSON(doc)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(j, &generic); err != nil {
		return nil, err
	}
	w := &keyWalker{}
	w.walk(generic, reflect.TypeOf(crdDoc{}), "", false)
	if len(w.unknown) > 0 {
		prefix := ""
		if k, ok := lookup(generic, "spec", "names", "kind").(string); ok && k != "" {
			prefix = k + ": "
		}
		return nil, fmt.Errorf("%s%s; the crd extractor reads only the fields its page shows", prefix, strings.Join(w.unknown, "; "))
	}
	d := &crdDoc{}
	if err := strictJSON(j, d); err != nil {
		return nil, err
	}
	if p := d.Spec.PreserveUnknownFields; p != nil && *p {
		return nil, fmt.Errorf("spec.preserveUnknownFields: true is not supported; an apiextensions.k8s.io/v1 CRD prunes unknown fields")
	}
	return d, nil
}

// lookup follows keys through nested JSON objects.
func lookup(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// keyWalker walks a document's JSON alongside the struct that decodes it
// and records every key the struct does not name, with its path: the
// document path ("spec.conversion"), or inside the schema the field path
// the page uses ("spec.module.path"; "the object" for the root).
type keyWalker struct{ unknown []string }

var (
	propsType = reflect.TypeOf(props{})
	rawType   = reflect.TypeOf(json.RawMessage{})
)

func (w *keyWalker) walk(v any, t reflect.Type, at string, inSchema bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == rawType:
	case t == reflect.TypeOf(orArray{}):
		w.walk(v, propsType, child(at, "[]"), true)
	case t == reflect.TypeOf(orBool{}):
		w.walk(v, propsType, child(at, ".<key>"), true)
	case t.Kind() == reflect.Slice:
		w.walkSlice(v, t, at, inSchema)
	case t.Kind() == reflect.Map && t.Elem() == propsType:
		m, _ := v.(map[string]any)
		for _, name := range sortedKeys(m) {
			w.walk(m[name], propsType, child(at, "."+name), true)
		}
	case t.Kind() == reflect.Struct:
		w.walkStruct(v, t, at, inSchema)
	}
}

func (w *keyWalker) walkSlice(v any, t reflect.Type, at string, inSchema bool) {
	s, _ := v.([]any)
	for i, e := range s {
		next := at
		if !inSchema {
			next = fmt.Sprintf("%s[%d]", at, i)
		}
		w.walk(e, t.Elem(), next, inSchema)
	}
}

func (w *keyWalker) walkStruct(v any, t reflect.Type, at string, inSchema bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	fields := map[string]reflect.StructField{}
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		fields[name] = t.Field(i)
	}
	for _, key := range sortedKeys(m) {
		f, ok := fields[key]
		switch {
		case !ok:
			w.unknown = append(w.unknown, fmt.Sprintf("%s: unknown field %q", unknownAt(at, key, inSchema), key))
		case key == "openAPIV3Schema":
			w.walk(m[key], f.Type, "", true)
		case inSchema:
			w.walk(m[key], f.Type, at, true)
		default:
			w.walk(m[key], f.Type, child(at, "."+key), false)
		}
	}
}

// unknownAt is where an unknown key is reported: the key's own document
// path, or inside the schema the node's field path.
func unknownAt(at, key string, inSchema bool) string {
	switch {
	case !inSchema:
		return child(at, "."+key)
	case at == "":
		return "the object"
	}
	return at
}

// child extends a path: a document path with ".key", a schema path with
// ".name", "[]" or ".<key>" (no leading dot at the root).
func child(at, seg string) string {
	if at == "" {
		return strings.TrimPrefix(seg, ".")
	}
	return at + seg
}
