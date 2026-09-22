// Command carrierproxy demonstrates Login against a demo site using
// credentials from the environment. See README.md for setup and usage.
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
	// demoLoginURL is a public site built for browser-automation practice;
	// see README.md. browser.NewClient takes any login URL, this is just
	// the demo.
	demoLoginURL = "https://the-internet.herokuapp.com/login"

	// envUsername and envPassword name the environment variables the
	// credentials are read from.
	envUsername = "CARRIERPROXY_USERNAME"
	envPassword = "CARRIERPROXY_PASSWORD"
)

func main() {
	if err := run(os.Getenv, browser.NewClient(demoLoginURL), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// run reads the credentials with getenv, performs one Login against
// provider and reports success on out, returning any failure for main to
// report. Everything process-level (the real environment, the real
// browser, stdout, os.Exit) is injected or left to main so run itself can
// be tested.
func run(getenv func(string) string, provider carrierproxy.PolicyProvider, out io.Writer) error {
	username, password := getenv(envUsername), getenv(envPassword)
	if username == "" || password == "" {
		return fmt.Errorf("%w: set %s and %s", carrierproxy.ErrInvalidCredentials, envUsername, envPassword)
	}

	if err := provider.Login(username, password); err != nil {
		return fmt.Errorf("login to %s failed: %w", demoLoginURL, err)
	}

	_, err := fmt.Fprintf(out, "login to %s succeeded as %s\n", demoLoginURL, username)
	return err
}
