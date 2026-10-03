package crd

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// sample returns the kind's example: the first document of the kind in the
// file kubebuilder names <group>_<version>_<kind>.yaml under o.Samples,
// re-encoded without comments and with the configured labels stripped. It
// returns nil when the file or the document is missing, or when the
// document holds a HideSamplesMatching string. No other file is read.
func sample(o Options, d *crdDoc) (*Sample, error) {
	dir, err := within(o.Root, o.Samples)
	if err != nil {
		return nil, err
	}
	group, version, kind := d.Spec.Group, d.Spec.Versions[0].Name, d.Spec.Names.Kind
	name := fmt.Sprintf("%s_%s_%s.yaml", group, version, strings.ToLower(kind))
	rel := relPath(o.Samples, name)
	data, err := os.ReadFile(filepath.Join(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, doc := range splitDocuments(data) {
		var obj map[string]any
		if err := yaml.Unmarshal(doc, &obj); err != nil {
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
