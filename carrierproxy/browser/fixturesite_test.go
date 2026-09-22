package browser

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

// flash is the login result banner data testdata/login.html renders.
type flash struct{ Class, Message string }

// fixtureSite is a minimal, local stand-in for a real carrier site: a login
// form served from testdata/ that accepts exactly one username/password
// pair. It exists so TestLoginIntegration can drive a real browser without
// depending on any external site.
type fixtureSite struct {
	username string
	password string
	loginTpl *template.Template
}

// newFixtureSite starts a fixture site that accepts username/password and
// registers its shutdown with t.Cleanup.
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

// handleLogin renders the login form and, on POST, checks the submitted
// credentials, rendering a success or error banner.
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
