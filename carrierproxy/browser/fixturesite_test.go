package browser

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// fixturePolicies is what the fixture site's /policies page lists.
var fixturePolicies = []carrierproxy.Policy{
	{CarrierID: "CARRIER-1", PolicyNumber: "POL-100"},
	{CarrierID: "CARRIER-2", PolicyNumber: "POL-200"},
}

// fixtureDocumentKey is the only downloadKey the fixture site's
// /download endpoint accepts; its content is testdata/<fixtureDocumentKey>.
const fixtureDocumentKey = "policy-42.txt"

// flash is the login result banner data testdata/login.html renders.
type flash struct{ Class, Message string }

// fixtureSite is a minimal, local stand-in for a real carrier site: a
// login form, a cookie-gated policies table, and a cookie-gated document
// download endpoint, all served from testdata/. It exists so
// TestLoginIntegration can drive a real browser without depending on any
// external site.
type fixtureSite struct {
	username, password    string
	loginTpl, policiesTpl *template.Template

	mu       sync.Mutex
	sessions map[string]bool
}

// newFixtureSite starts a fixture site that accepts username/password,
// and registers its shutdown with t.Cleanup.
func newFixtureSite(t *testing.T, username, password string) *httptest.Server {
	t.Helper()

	site := &fixtureSite{
		username:    username,
		password:    password,
		loginTpl:    template.Must(template.ParseFiles("testdata/login.html")),
		policiesTpl: template.Must(template.ParseFiles("testdata/policies.html")),
		sessions:    map[string]bool{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", site.handleLogin)
	mux.HandleFunc("/policies", site.handlePolicies)
	mux.HandleFunc("/download/", site.handleDownload)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// handleLogin renders the login form, and on POST checks the submitted
// credentials, starting a session on success.
func (s *fixtureSite) handleLogin(w http.ResponseWriter, r *http.Request) {
	var data struct{ Flash *flash }
	if r.Method == http.MethodPost {
		data.Flash = s.tryLogin(w, r)
	}
	_ = s.loginTpl.Execute(w, data)
}

// tryLogin checks the submitted credentials, starts a session on success,
// and returns the flash to render either way.
func (s *fixtureSite) tryLogin(w http.ResponseWriter, r *http.Request) *flash {
	username, password := r.FormValue("username"), r.FormValue("password")
	switch {
	case username != s.username:
		return &flash{"error", "Your username is invalid!"}
	case password != s.password:
		return &flash{"error", "Your password is invalid!"}
	default:
		http.SetCookie(w, &http.Cookie{Name: "session", Value: s.startSession(), Path: "/"})
		return &flash{"success", "You logged into a secure area!"}
	}
}

// startSession records and returns a new valid session token.
func (s *fixtureSite) startSession() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	token := fmt.Sprintf("session-%d", len(s.sessions)+1)
	s.sessions[token] = true
	return token
}

// authenticated reports whether r carries a session cookie this site
// started.
func (s *fixtureSite) authenticated(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[c.Value]
}

// handlePolicies serves the policies table to an authenticated session.
func (s *fixtureSite) handlePolicies(w http.ResponseWriter, r *http.Request) {
	if !s.authenticated(r) {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	}
	_ = s.policiesTpl.Execute(w, struct{ Policies []carrierproxy.Policy }{fixturePolicies})
}

// handleDownload serves the fixture document to an authenticated session.
func (s *fixtureSite) handleDownload(w http.ResponseWriter, r *http.Request) {
	if !s.authenticated(r) {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/download/")
	if key != fixtureDocumentKey {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "testdata/"+fixtureDocumentKey)
}
