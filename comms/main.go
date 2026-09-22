// Command comms runs the comms REST API. Each route accepts a JSON POST and
// sends a templated email through sendgrid.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/handlers"
)

// Built-in defaults, each overridable through the environment variable
// named next to it in loadConfig.
const (
	defaultAddr             = ":8090"
	defaultEmailAPIKey      = "FOOBAR123XYZ"
	defaultEmailFromName    = "Foo Bar"
	defaultEmailFromAddress = "foo@bar.com"
)

// config is the server's runtime configuration.
type config struct {
	addr  string
	email sendgrid.Config
}

// loadConfig builds the configuration from the environment, falling back
// to the built-in defaults for any unset or empty variable. getenv is
// injected so tests can drive it without touching the process environment.
func loadConfig(getenv func(string) string) config {

	pick := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	return config{
		addr: pick("COMMS_ADDR", defaultAddr),
		email: sendgrid.Config{
			APIKey:      pick("COMMS_SENDGRID_API_KEY", defaultEmailAPIKey),
			FromName:    pick("COMMS_EMAIL_FROM_NAME", defaultEmailFromName),
			FromAddress: pick("COMMS_EMAIL_FROM_ADDRESS", defaultEmailFromAddress),
		},
	}
}

// newMux builds the comms API's route table against the given email
// provider. It is exercised directly by an httptest-backed integration
// test, without a real network listener or sendgrid credentials.
func newMux(emailsvc email.MailProvider) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/comms/add-policy-vehicle", handlers.AddPolicyVehicle(emailsvc))
	mux.HandleFunc("/api/comms/add-policy-driver", handlers.AddPolicyDriver(emailsvc))
	mux.HandleFunc("/api/comms/add-policy-address", handlers.AddPolicyAddress(emailsvc))
	mux.HandleFunc("/api/comms/add-policy-coverage", handlers.AddPolicyCoverage(emailsvc))
	return mux
}

// newServer wraps h in an http.Server listening on addr with read, write
// and idle timeouts, so a slow or stalled client cannot pin a connection
// indefinitely.
func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// run wires the sendgrid-backed mux into a server and serves it until the
// listener fails; it only ever returns a non-nil error.
func run(cfg config) error {

	// initialize the email service
	emailsvc := sendgrid.NewSvc(cfg.email)

	// start the api server
	srv := newServer(cfg.addr, newMux(emailsvc))
	log.Printf("starting server on %s...", srv.Addr)
	return srv.ListenAndServe()
}

func main() {
	log.Fatal(run(loadConfig(os.Getenv)))
}
