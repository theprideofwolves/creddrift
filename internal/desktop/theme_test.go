package desktop

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestThemePreferenceAndSafeReturn(t *testing.T) {
	for _, target := range []string{"/setup?token=old", "https://evil.test/", "//evil.test/", "/export?token=old"} {
		r := httptest.NewRequest("POST", "http://localhost/theme?token=t", strings.NewReader(url.Values{"theme": {"dark"}, "return": {target}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		SetTheme(w, r, "t")
		if w.Code != 303 {
			t.Fatal(w.Code)
		}
		u, err := url.Parse(w.Header().Get("Location"))
		if err != nil || u.Host != "" || u.Query().Get("token") != "t" {
			t.Fatal("unsafe return")
		}
		if target == "/setup?token=old" && u.Path != "/setup" {
			t.Fatal("setup not preserved")
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatal("preference cookie policy")
		}
		next := httptest.NewRequest("GET", "http://localhost/", nil)
		next.AddCookie(cookies[0])
		if Theme(next) != "dark" || !strings.Contains(string(ThemeControl(next, "t")), "Light mode") {
			t.Fatal("preference lost")
		}
	}
	for _, method := range []string{"GET", "POST"} {
		w := httptest.NewRecorder()
		SetTheme(w, httptest.NewRequest(method, "http://localhost/theme", strings.NewReader("theme=invalid")), "t")
		if w.Code < 400 || len(w.Result().Cookies()) != 0 {
			t.Fatal("invalid change accepted")
		}
	}
}
