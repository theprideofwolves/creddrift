package ticket

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestEquivalentOriginsCannotDuplicateCreate(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	s.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"key":"SEC-1"}`)), Header: make(http.Header)}, nil
	})}
	for _, origin := range []string{"https://jira.example.test", "https://JIRA.EXAMPLE.TEST:443/"} {
		if _, err := s.Handoff(context.Background(), Config{origin, "test-user", "synthetic-test-token", "SEC", "Task"}, "fixture", "Review", "Metadata only", ""); err != nil {
			t.Fatal(err)
		}
	}
	if len(paths) != 2 || paths[1] != "/rest/api/3/issue/SEC-1/comment" {
		t.Fatal("equivalent origins caused duplicate creation")
	}
}

func TestLegacyOriginMappingIsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tickets.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Records["https://JIRA.EXAMPLE.TEST:443|fixture"] = Record{Key: "SEC-1", State: "linked"}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Records["https://jira.example.test|fixture"].Key != "SEC-1" {
		t.Fatal("legacy mapping lost")
	}
	s.Records["https://jira.example.test|fixture"] = Record{Key: "SEC-2", State: "linked"}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Records["https://jira.example.test|fixture"].State != "uncertain" {
		t.Fatal("conflicting legacy mappings not blocked")
	}
}
