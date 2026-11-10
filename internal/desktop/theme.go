package desktop

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/url"
)

// Theme stores only a display preference, independent of application data.
func Theme(r *http.Request) string {
	if c, err := r.Cookie("creddrift-theme"); err == nil && c.Value == "dark" {
		return "dark"
	}
	return "light"
}

func ThemeControl(r *http.Request, token string) template.HTML {
	next, label := "dark", "Dark mode"
	if Theme(r) == "dark" {
		next, label = "light", "Light mode"
	}
	return template.HTML(fmt.Sprintf(`<form class="theme-control" method="post" action="/theme?token=%s"><input type="hidden" name="theme" value="%s"><input type="hidden" name="return" value="%s"><button aria-label="Switch to %s">%s</button></form>`, html.EscapeString(url.QueryEscape(token)), next, html.EscapeString(r.URL.RequestURI()), label, label))
}

// SetTheme must run behind the same token, method and origin guards as other forms.
func SetTheme(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil || (r.PostForm.Get("theme") != "light" && r.PostForm.Get("theme") != "dark") {
		http.Error(w, "Choose light or dark mode", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "creddrift-theme", Value: r.PostForm.Get("theme"), Path: "/", MaxAge: 31536000, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	target := &url.URL{Path: "/"}
	if u, err := url.Parse(r.PostForm.Get("return")); err == nil && u.Host == "" && u.Scheme == "" {
		switch u.Path {
		case "/", "/welcome", "/setup":
			target.Path = u.Path
			target.RawQuery = u.RawQuery
		}
	}
	q := target.Query()
	q.Set("token", token)
	target.RawQuery = q.Encode()
	http.Redirect(w, r, target.String(), http.StatusSeeOther)
}

// Shared by onboarding and findings. No scripts, external fonts or remote assets.
const Style = `
:root{color-scheme:light;--bg:#fff;--surface:#fff;--text:#181818;--muted:#595959;--border:#d6d6d6;--accent:#006b58;--soft:#eef7f4;--on-accent:#fff}
:root[data-theme="dark"]{color-scheme:dark;--bg:#000;--surface:#000;--text:#ededed;--muted:#aaa;--border:#383838;--accent:#66a69b;--soft:#0b201a;--on-accent:#001e16}
*{box-sizing:border-box}
body{font:16px/1.6 system-ui,sans-serif;background:var(--bg);color:var(--text);max-width:1060px;margin:32px auto;padding:0 24px 40px}
h1,h2,h3{line-height:1.25;font-weight:650;letter-spacing:-.025em}
h1{font-size:32px;margin:24px 0 12px}
h2{font-size:21px}
h3{font-size:17px}
p{margin:12px 0}
small{color:var(--muted);line-height:1.6}
a{color:var(--accent);text-underline-offset:3px}
a:hover{text-decoration-thickness:2px}
section{border-top:1px solid var(--border);padding:22px 0;margin:24px 0}
section section{margin:16px 0;padding:12px 0}
label{display:block;margin:16px 0;font-weight:550}
label small{font-weight:400}
input,textarea,select{display:block;width:100%;max-width:760px;background:var(--surface);color:var(--text);border:1px solid var(--border);padding:10px 12px;border-radius:3px;margin:6px 0;font:inherit}
input[type=hidden]{display:none}
textarea{min-height:120px}
button{background:var(--accent);color:var(--on-accent);font:600 14px/1.4 system-ui,sans-serif;padding:10px 15px;border:1px solid var(--accent);border-radius:3px;cursor:pointer;margin:6px 6px 6px 0;min-height:42px}
button:hover{filter:brightness(.92)}
:focus-visible{outline:3px solid var(--accent);outline-offset:3px}
.theme-control{display:flex;justify-content:flex-end;margin:0}
.theme-control button{background:transparent;color:var(--accent)}
.notice{padding:12px 16px;background:var(--soft);border-left:3px solid var(--accent)}
.choices{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:32px}
nav{display:flex;flex-wrap:wrap;gap:8px 14px;border-bottom:1px solid var(--border);padding:12px 0}
article{padding:8px 0 24px}
article p,article small,pre{overflow-wrap:anywhere}
pre{white-space:pre-wrap}
hr{border:0;border-top:1px solid var(--border)}
summary{cursor:pointer;color:var(--accent);padding:12px 0}
td,th{padding:10px;text-align:left;border-bottom:1px solid var(--border);overflow-wrap:anywhere}
table{border-collapse:collapse;width:100%}
.scroll{overflow:auto}
@media(max-width:600px){body{margin:16px auto;padding:0 16px 24px}
.choices{grid-template-columns:1fr;gap:0}
h1{font-size:28px}
button{max-width:100%}}
`
