package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
)

// TrustOptions says where the Sigstore trusted root comes from.
type TrustOptions struct {
	// File, when set, is a trusted_root.json to use as it is (tests and
	// air-gapped hosts).
	File string
	// CacheDir holds the TUF metadata and the last trusted_root.json.
	CacheDir string
	// Offline uses the cached trusted root without refreshing it.
	Offline bool
	// Warn receives a warning, such as expired cached metadata.
	Warn func(string)
}

const trustedRootFile = "trusted_root.json"

// TrustedRoot loads the Sigstore public-good trusted root: from File, from
// the cache when offline, else through TUF (caching the result).
func TrustedRoot(o TrustOptions) (root.TrustedMaterial, error) {
	if o.File != "" {
		tr, err := root.NewTrustedRootFromPath(o.File)
		if err != nil {
			return nil, fmt.Errorf("reading the trusted root %s: %w", o.File, err)
		}
		return tr, nil
	}
	cached := filepath.Join(o.CacheDir, trustedRootFile)
	if o.Offline {
		tr, err := root.NewTrustedRootFromPath(cached)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no cached Sigstore trusted root at %s: run pull once with network", cached)
		}
		if err != nil {
			return nil, fmt.Errorf("reading the cached trusted root %s: %w", cached, err)
		}
		if exp, ok := tufExpiry(filepath.Join(o.CacheDir, "tuf")); ok && time.Now().After(exp) && o.Warn != nil {
			o.Warn(fmt.Sprintf("the cached Sigstore TUF metadata expired on %s; verifying with the cached trusted root anyway (each signature is checked at its own time)", exp.Format(time.RFC3339)))
		}
		return tr, nil
	}
	opts := tuf.DefaultOptions().WithCachePath(filepath.Join(o.CacheDir, "tuf"))
	c, err := tuf.New(opts)
	if err != nil {
		return nil, fmt.Errorf("fetching the Sigstore trusted root: %w", err)
	}
	raw, err := c.GetTarget(trustedRootFile)
	if err != nil {
		return nil, fmt.Errorf("fetching the Sigstore trusted root: %w", err)
	}
	tr, err := root.NewTrustedRootFromJSON(raw)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(o.CacheDir, 0o750); err == nil {
		_ = os.WriteFile(cached, raw, 0o600)
	}
	return tr, nil
}

// tufExpiry reads the expiry of the cached TUF timestamp metadata,
// <dir>/<repository>/timestamp.json.
func tufExpiry(dir string) (expires time.Time, found bool) {
	paths, _ := filepath.Glob(filepath.Join(dir, "*", "timestamp.json"))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var meta struct {
			Signed struct {
				Expires time.Time `json:"expires"`
			} `json:"signed"`
		}
		if json.Unmarshal(b, &meta) == nil && !meta.Signed.Expires.IsZero() {
			expires, found = meta.Signed.Expires, true
		}
	}
	return expires, found
}
