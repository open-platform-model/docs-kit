package verify_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/verify"
)

// TestOfflineWithoutCachedRoot: an offline pull with no cached trusted
// root fails, naming the cache and the fix, and fetches nothing.
func TestOfflineWithoutCachedRoot(t *testing.T) {
	dir := t.TempDir()
	_, err := verify.TrustedRoot(verify.TrustOptions{CacheDir: dir, Offline: true})
	if err == nil || !strings.Contains(err.Error(), "no cached Sigstore trusted root") || !strings.Contains(err.Error(), dir) {
		t.Fatalf("err = %v", err)
	}
}

func cacheWith(t *testing.T, expires string) string {
	t.Helper()
	dir := t.TempDir()
	root, err := os.ReadFile("testdata/public-good-trusted-root.json")
	if err != nil {
		t.Fatal(err)
	}
	ts := filepath.Join(dir, "tuf", "tuf-repo-cdn.sigstore.dev", "timestamp.json")
	if err := os.MkdirAll(filepath.Dir(ts), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trusted_root.json"), root, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ts, []byte(`{"signed":{"expires":"`+expires+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestOfflineExpiredRootWarns: an expired cached TUF timestamp warns, and
// the cached root still verifies the real spike signature.
func TestOfflineExpiredRootWarns(t *testing.T) {
	var warnings []string
	tm, err := verify.TrustedRoot(verify.TrustOptions{CacheDir: cacheWith(t, "2020-01-01T00:00:00Z"), Offline: true,
		Warn: func(s string) { warnings = append(warnings, s) }})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "expired on 2020-01-01T00:00:00Z") {
		t.Fatalf("warnings %q", warnings)
	}
	v, err := verify.New(tm)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Bundle(spikeBundle(t), spikeDigest, spikePolicy()); err != nil {
		t.Fatalf("an expired cache refused a valid signature: %v", err)
	}
	warnings = nil
	if _, err := verify.TrustedRoot(verify.TrustOptions{CacheDir: cacheWith(t, "2999-01-01T00:00:00Z"), Offline: true,
		Warn: func(s string) { warnings = append(warnings, s) }}); err != nil || len(warnings) != 0 {
		t.Fatalf("fresh cache: %v %q", err, warnings)
	}
}
