package main

import (
	"log"
	"net"
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

	// start the api server
	ln, err := net.Listen("tcp", ":8090")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("starting server...")
	log.Fatal(serve(ln, newEmailSvc()))
}

// newEmailSvc initializes the sendgrid-backed email service from the app
// config.
func newEmailSvc() *sendgrid.Client {
	return sendgrid.NewSvc(sendgrid.Config{
		APIKey:      cfgEmailAPIKey,
		FromName:    cfgEmailFromName,
		FromAddress: cfgEmailFromAddress,
	})
}

// serve runs the comms API on ln until ln is closed or the server fails,
// always returning a non-nil error. It takes a listener rather than an
// address so the end-to-end test can bind an ephemeral port.
func serve(ln net.Listener, emailsvc email.MailProvider) error {
	return http.Serve(ln, newMux(emailsvc))
}
