package main

import (
	"log"
	"net/http"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/handlers"
)

var (
	cfgEmailAPIKey      string
	cfgEmailFromName    string
	cfgEmailFromAddress string
)

func init() {

	// load the app config
	cfgEmailAPIKey = "FOOBAR123XYZ"
	cfgEmailFromName = "Foo Bar"
	cfgEmailFromAddress = "foo@bar.com"
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

func main() {

	// initialize the email service
	emailsvc := sendgrid.NewSvc(sendgrid.Config{
		APIKey:      cfgEmailAPIKey,
		FromName:    cfgEmailFromName,
		FromAddress: cfgEmailFromAddress,
	})

	// start the api server
	log.Print("starting server...")
	log.Fatal(http.ListenAndServe(":8090", newMux(emailsvc)))
}
