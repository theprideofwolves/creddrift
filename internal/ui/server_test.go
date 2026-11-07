package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalActionBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, host, origin, method string
		want                       int
	}{
		{"local", "localhost:8080", "http://localhost:8080", "POST", 204}, {"foreign-host", "attacker.test:8080", "", "POST", 403}, {"foreign-origin", "localhost:8080", "https://attacker.test", "POST", 403}, {"wrong-method", "localhost:8080", "", "GET", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := localOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
			r := httptest.NewRequest(tc.method, "http://localhost:8080/api/rotate", nil)
			r.Host = tc.host
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d want %d", w.Code, tc.want)
			}
		})
	}
}
