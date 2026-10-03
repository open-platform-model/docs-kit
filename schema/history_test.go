package schema

import (
	"strings"
	"testing"
)

// history is a valid history.json around one member, backup, first in the
// floor 4.5 and given a required field in 4.6.
const history = `{
  "schema": "docs.opmodel.dev/history/v1",
  "project": "catalog-opm",
  "tool": "0.3.0",
  "floor": "4.5",
  "segments": %s,
  "compared": [
    {"from": "4.5", "to": "4.6", "mode": "full"},
    {"from": "4.6", "to": "edge", "mode": "paths"}
  ],
  "members": {
    "opmodel.dev/catalogs/opm/traits/backup@v1alpha1": {
      "kind": "trait", "name": "backup", "apiVersion": "v1alpha1",
      "first": "4.5", "firstIsFloor": true, "in": ["4.5", "4.6", "edge"],
      "changes": {"4.6": [{"op": "presence", "path": "retention.daily", "from": "optional", "to": "required"}]}
    }
  },
  "removed": {
    "edge": [{"fqn": "opmodel.dev/catalogs/opm/traits/old@v1alpha1", "kind": "trait", "name": "old", "apiVersion": "v1alpha1", "lastIn": "4.6", "page": "traits/old"}]
  },
  "lineage": {"trait/backup": {"4.5": ["v1alpha1"], "4.6": ["v1alpha1"], "edge": ["v1alpha1"]}}
}`

func TestHistoryValidates(t *testing.T) {
	ok := strings.Replace(history, "%s", `["4.5", "4.6", "edge"]`, 1)
	if _, err := ValidateJSON("#History", "history.json", []byte(ok)); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"one segment":   strings.Replace(history, "%s", `["4.5"]`, 1),
		"bad mode":      strings.Replace(ok, `"mode": "paths"`, `"mode": "none"`, 1),
		"bad op":        strings.Replace(ok, `"op": "presence"`, `"op": "renamed"`, 1),
		"edge floor":    strings.Replace(ok, `"floor": "4.5"`, `"floor": "edge"`, 1),
		"extra key":     strings.Replace(ok, `"tool": "0.3.0",`, `"tool": "0.3.0", "at": "now",`, 1),
		"presence":      strings.Replace(ok, `"to": "required"`, `"to": "mandatory"`, 1),
		"null presence": strings.Replace(ok, `"to": "required"`, `"to": null`, 1),
		"bad page":      strings.Replace(ok, `"page": "traits/old"`, `"page": "../traits/old"`, 1),
	}
	for name, doc := range cases {
		if _, err := ValidateJSON("#History", "history.json", []byte(doc)); err == nil {
			t.Errorf("%s: validated", name)
		}
	}
}
