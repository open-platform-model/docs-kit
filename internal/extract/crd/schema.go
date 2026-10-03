package crd

import (
	"bytes"
	"encoding/json"
)

// The fields of an apiextensions.k8s.io/v1 CustomResourceDefinition the
// extractor reads. A local struct rather than the Kubernetes types, which
// would add the Kubernetes API machinery to the module graph for a handful
// of fields.

type crdDoc struct {
	APIVersion string  `json:"apiVersion"`
	Kind       string  `json:"kind"`
	Spec       crdSpec `json:"spec"`
}

type crdSpec struct {
	Group    string       `json:"group"`
	Scope    string       `json:"scope"`
	Names    crdNames     `json:"names"`
	Versions []crdVersion `json:"versions"`
}

type crdNames struct {
	Kind       string   `json:"kind"`
	Plural     string   `json:"plural"`
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
		Name     string `json:"name"`
		Type     string `json:"type"`
		JSONPath string `json:"jsonPath"`
		Priority int    `json:"priority"`
	} `json:"additionalPrinterColumns"`
}

// props is one OpenAPI v3 schema node.
type props struct {
	Type                 string            `json:"type"`
	Format               string            `json:"format"`
	Description          string            `json:"description"`
	Properties           map[string]props  `json:"properties"`
	Required             []string          `json:"required"`
	Items                *orArray          `json:"items"`
	AdditionalProperties *orBool           `json:"additionalProperties"`
	Default              json.RawMessage   `json:"default"`
	Enum                 []json.RawMessage `json:"enum"`
	Pattern              string            `json:"pattern"`
	MinLength            *int64            `json:"minLength"`
	MaxLength            *int64            `json:"maxLength"`
	MinItems             *int64            `json:"minItems"`
	MaxItems             *int64            `json:"maxItems"`
	MinProperties        *int64            `json:"minProperties"`
	MaxProperties        *int64            `json:"maxProperties"`
	Minimum              *float64          `json:"minimum"`
	Maximum              *float64          `json:"maximum"`
	ExclusiveMinimum     bool              `json:"exclusiveMinimum"`
	ExclusiveMaximum     bool              `json:"exclusiveMaximum"`
	UniqueItems          bool              `json:"uniqueItems"`
	Nullable             bool              `json:"nullable"`
	AllOf                []json.RawMessage `json:"allOf"`
	OneOf                []json.RawMessage `json:"oneOf"`
	AnyOf                []json.RawMessage `json:"anyOf"`
	Not                  json.RawMessage   `json:"not"`
	XIntOrString         bool              `json:"x-kubernetes-int-or-string"`
	XPreserveUnknown     *bool             `json:"x-kubernetes-preserve-unknown-fields"`
	XListType            *string           `json:"x-kubernetes-list-type"`
	XListMapKeys         []string          `json:"x-kubernetes-list-map-keys"`
	XValidations         []struct {
		Rule    string `json:"rule"`
		Message string `json:"message"`
	} `json:"x-kubernetes-validations"`
}

// orArray is items: one schema, or a list of them, which the reference
// does not descend into.
type orArray struct{ Schema *props }

func (o *orArray) UnmarshalJSON(b []byte) error {
	if bytes.HasPrefix(bytes.TrimSpace(b), []byte("[")) {
		return nil
	}
	o.Schema = &props{}
	return json.Unmarshal(b, o.Schema)
}

// orBool is additionalProperties: a schema, or true or false.
type orBool struct{ Schema *props }

func (o *orBool) UnmarshalJSON(b []byte) error {
	t := bytes.TrimSpace(b)
	if bytes.Equal(t, []byte("true")) || bytes.Equal(t, []byte("false")) {
		return nil
	}
	o.Schema = &props{}
	return json.Unmarshal(b, o.Schema)
}

func (p props) preserveUnknown() bool { return p.XPreserveUnknown != nil && *p.XPreserveUnknown }

func (p props) listType(t string) bool { return p.XListType != nil && *p.XListType == t }
