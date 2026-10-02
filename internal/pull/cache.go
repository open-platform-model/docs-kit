package pull

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
)

// Cache keeps fetched blobs by digest, and each bundle's verified
// signature, so a frozen pull can run offline.
type Cache struct {
	Dir string
}

// DefaultCacheDir is $XDG_CACHE_HOME/opm-docs, else ~/.cache/opm-docs.
func DefaultCacheDir() string {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "opm-docs")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".cache", "opm-docs")
	}
	return filepath.Join(os.TempDir(), "opm-docs-cache")
}

func (c Cache) blobPath(d digest.Digest) string {
	return filepath.Join(c.Dir, "blobs", d.Algorithm().String(), d.Encoded())
}

func (c Cache) sigPath(d digest.Digest) string {
	return filepath.Join(c.Dir, "signatures", d.Algorithm().String(), d.Encoded()+".sigstore.json")
}

// Blob returns a cached blob, checking its digest; ok is false when it is
// not cached.
func (c Cache) Blob(d digest.Digest) (data []byte, ok bool, err error) {
	b, err := os.ReadFile(c.blobPath(d))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if digest.FromBytes(b) != d {
		return nil, false, fmt.Errorf("cached blob %s is corrupt; remove %s", d, c.blobPath(d))
	}
	return b, true, nil
}

// PutBlob caches a blob.
func (c Cache) PutBlob(d digest.Digest, b []byte) error {
	return write(c.blobPath(d), b)
}

// Signature returns the cached Sigstore bundle that verified d.
func (c Cache) Signature(d digest.Digest) (data []byte, ok bool, err error) {
	b, err := os.ReadFile(c.sigPath(d))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return b, err == nil, err
}

// PutSignature caches the Sigstore bundle that verified d.
func (c Cache) PutSignature(d digest.Digest, b []byte) error {
	return write(c.sigPath(d), b)
}

func write(p string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
