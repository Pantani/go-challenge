// Command carrierproxy demonstrates Login against a demo site using
// credentials from the environment. See README.md for setup and usage.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy/browser"
)

// demoLoginURL is a public site built for browser-automation practice; see
// README.md. browser.NewClient takes any login URL, this is just the demo.
const demoLoginURL = "https://the-internet.herokuapp.com/login"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run wires up a browser.Client and performs one login, returning any
// failure for main to report; kept separate from main so the only os.Exit
// path in this program is the one in main itself.
func run() error {
	username := os.Getenv("CARRIERPROXY_USERNAME")
	password := os.Getenv("CARRIERPROXY_PASSWORD")

	client := browser.NewClient(demoLoginURL)
	if err := client.Login(username, password); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	log.Println("login succeeded")
	return nil
}
