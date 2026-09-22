package browser

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

type flash struct{ Class, Message string }

type fixtureSite struct {
	username string
	password string
	loginTpl *template.Template
}

func newFixtureSite(t *testing.T, username, password string) *httptest.Server {
	t.Helper()
	site := fixtureSite{
		username: username,
		password: password,
		loginTpl: template.Must(template.ParseFiles("testdata/login.html")),
	}
	server := httptest.NewServer(http.HandlerFunc(site.handleLogin))
	t.Cleanup(server.Close)
	return server
}

func (s fixtureSite) handleLogin(w http.ResponseWriter, r *http.Request) {
	var data struct{ Flash *flash }
	if r.Method == http.MethodPost {
		if r.FormValue("username") == s.username && r.FormValue("password") == s.password {
			data.Flash = &flash{Class: "success", Message: "You logged into a secure area!"}
		} else {
			data.Flash = &flash{Class: "error", Message: "Your credentials are invalid."}
		}
	}
	_ = s.loginTpl.Execute(w, data)
}
