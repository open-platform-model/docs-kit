package crd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// sample returns the kind's example: the first document of the kind in the
// file kubebuilder names <group>_<version>_<kind>.yaml under Samples,
// re-encoded without comments and with the configured labels stripped. It
// returns nil when the file or the document is missing, or when the
// document holds a HideSamplesMatching string. No other file is read.
func (x *extraction) sample(d *crdDoc) (*Sample, error) {
	o := x.o
	dir, err := within(o.Root, o.Samples)
	if err != nil {
		return nil, err
	}
	group, version, kind := d.Spec.Group, d.Spec.Versions[0].Name, d.Spec.Names.Kind
	name := fmt.Sprintf("%s_%s_%s.yaml", group, version, strings.ToLower(kind))
	rel := relPath(o.Samples, name)
	data, err := x.read(filepath.Join(dir, name), rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, doc := range splitDocuments(data) {
		obj, err := decodeSample(doc)
		if err != nil {
			return nil, fmt.Errorf("%s: does not parse as YAML: %w", rel, err)
		}
		if obj["apiVersion"] != group+"/"+version || obj["kind"] != kind {
			continue
		}
		for _, h := range o.HideSamplesMatching {
			if bytes.Contains(doc, []byte(h)) {
				return nil, nil
			}
		}
		stripLabels(obj, o.StripLabels)
		out, err := yaml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		return &Sample{File: rel, YAML: string(out)}, nil
	}
	return nil, nil
}

// decodeSample decodes one YAML document into a map, numbers kept as
// written so a large integer survives the re-encoding.
func decodeSample(doc []byte) (map[string]any, error) {
	j, err := yaml.YAMLToJSON(doc)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// stripLabels removes each label of strip from obj's metadata.labels when
// its value matches, and labels itself when that empties it.
func stripLabels(obj map[string]any, strip map[string]string) {
	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		return
	}
	labels, ok := meta["labels"].(map[string]any)
	if !ok {
		return
	}
	for k, v := range strip {
		if labels[k] == v {
			delete(labels, k)
		}
	}
	if len(labels) == 0 {
		delete(meta, "labels")
	}
}
