// Package ocitest runs an in-process OCI registry for tests. Like GHCR it
// has no referrers API, so signatures are found through the fallback tag,
// and it pages tag lists with Link headers.
package ocitest

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
)

// Registry starts a registry for the test and returns its host:port.
func Registry(t testing.TB) string {
	t.Helper()
	return New(t).Host
}

// Start starts a registry and returns its host:port and a function that
// stops it, for a test that must prove it makes no network call.
func Start(t testing.TB) (host string, stop func()) {
	t.Helper()
	s := New(t)
	return s.Host, s.Stop
}

// Server is a running test registry that records the requests it serves.
type Server struct {
	Host string
	Stop func()

	mu       sync.Mutex
	requests []string // "<method> <path>"
}

// New starts a recording registry.
func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{}
	reg := paged(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.Method+" "+r.URL.Path)
		s.mu.Unlock()
		reg.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	s.Host, s.Stop = strings.TrimPrefix(srv.URL, "http://"), srv.Close
	return s
}

// BlobGets counts the GET requests for one blob digest.
func (s *Server) BlobGets(digest string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.requests {
		if strings.HasPrefix(r, "GET ") && strings.HasSuffix(r, "/blobs/"+digest) {
			n++
		}
	}
	return n
}

// paged serves tag lists in pages with Link headers, as GHCR does: the
// in-process registry neither honors "last" nor sends Link.
func paged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		n, _ := strconv.Atoi(q.Get("n"))
		if !strings.HasSuffix(r.URL.Path, "/tags/list") || n <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		all := r.Clone(r.Context())
		all.URL.RawQuery = ""
		all.RequestURI = ""
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, all)
		if rec.Code != http.StatusOK {
			w.WriteHeader(rec.Code)
			_, _ = w.Write(rec.Body.Bytes())
			return
		}
		var list struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &list)
		sort.Strings(list.Tags)
		last := q.Get("last")
		page := list.Tags[:0:0]
		for _, t := range list.Tags {
			if t > last && len(page) < n {
				page = append(page, t)
			}
		}
		if len(page) == n && page[n-1] != list.Tags[len(list.Tags)-1] {
			next := url.Values{"n": {strconv.Itoa(n)}, "last": {page[n-1]}}
			w.Header().Set("Link", fmt.Sprintf(`<%s?%s>; rel="next"`, r.URL.Path, next.Encode()))
		}
		list.Tags = page
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})
}
