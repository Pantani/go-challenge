// Command carrierproxy logs into a website's login form through the go-rod
// backed carrierproxy.PolicyProvider, reading the target URL and the
// credentials from the environment. See README.md for usage.
package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy/browser"
)

const (
	// envLoginURL names the environment variable holding the login page URL;
	// defaultLoginURL is used when it is unset.
	envLoginURL = "CARRIERPROXY_LOGIN_URL"

	// envUsername and envPassword name the environment variables the
	// credentials are read from.
	envUsername = "CARRIERPROXY_USERNAME"
	envPassword = "CARRIERPROXY_PASSWORD"

	// defaultLoginURL is a public practice site whose login form matches
	// browser.NewClient's default selectors. README.md lists its published
	// test credentials.
	defaultLoginURL = "https://the-internet.herokuapp.com/login"
)

func main() {
	newProvider := func(loginURL string) carrierproxy.PolicyProvider { return browser.NewClient(loginURL) }
	if err := run(os.Getenv, newProvider, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// run reads the login URL and credentials with getenv, performs one Login
// through the provider built by newProvider and reports the outcome on out.
// The environment, browser and stdout are injected so run can be tested.
func run(getenv func(string) string, newProvider func(string) carrierproxy.PolicyProvider, out io.Writer) error {
	loginURL := getenv(envLoginURL)
	if loginURL == "" {
		loginURL = defaultLoginURL
	}

	username, password := getenv(envUsername), getenv(envPassword)
	if username == "" || password == "" {
		return fmt.Errorf("%w: set %s and %s", carrierproxy.ErrInvalidCredentials, envUsername, envPassword)
	}

	if err := newProvider(loginURL).Login(username, password); err != nil {
		return fmt.Errorf("login to %s failed: %w", loginURL, err)
	}

	_, err := fmt.Fprintf(out, "login to %s succeeded as %s\n", loginURL, username)
	return err
}
