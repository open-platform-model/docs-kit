package crd

import (
	"bytes"
	"encoding/json"
	"fmt"

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
}

type crdSpec struct {
	Group    string       `json:"group"`
	Scope    string       `json:"scope"`
	Names    crdNames     `json:"names"`
	Versions []crdVersion `json:"versions"`
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
	Schema  *struct {
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

// decodeCRD decodes one YAML document strictly.
func decodeCRD(doc []byte) (*crdDoc, error) {
	j, err := yaml.YAMLToJSON(doc)
	if err != nil {
		return nil, err
	}
	d := &crdDoc{}
	if err := strictJSON(j, d); err != nil {
		return nil, fmt.Errorf("%w; the crd extractor reads only the CRD fields its page shows", err)
	}
	return d, nil
}
