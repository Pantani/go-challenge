package main

import (
	"log"
	"net/http"

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

func main() {

	// initialize the email service
	emailsvc := sendgrid.NewSvc(sendgrid.Config{
		APIKey:      cfgEmailAPIKey,
		FromName:    cfgEmailFromName,
		FromAddress: cfgEmailFromAddress,
	})

	// setup the api routes
	http.HandleFunc("/api/comms/add-policy-vehicle", handlers.AddPolicyVehicle(emailsvc))
	http.HandleFunc("/api/comms/add-policy-driver", handlers.AddPolicyDriver(emailsvc))
	http.HandleFunc("/api/comms/add-policy-address", handlers.AddPolicyAddress(emailsvc))
	http.HandleFunc("/api/comms/add-policy-coverage", handlers.AddPolicyCoverage(emailsvc))

	// start the api server
	log.Print("starting server...")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
