package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

func AddPolicyVehicle(emailsvc email.MailProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
			return
		}

		payload := AddPolicyVehicleReq{}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if err := emailsvc.Send([]string{payload.EmailTo}, payload.Message, email.TplAddPolicyVehicle); err != nil {
			http.Error(w, fmt.Sprintf("error sending email: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func AddPolicyDriver(emailsvc email.MailProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload := AddPolicyDriverReq{}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if err := emailsvc.Send([]string{payload.EmailTo}, payload.Message, email.TplAddPolicyDriver); err != nil {
			http.Error(w, fmt.Sprintf("error sending email: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func AddPolicyAddress(emailsvc email.MailProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload := AddPolicyAddressReq{}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if err := emailsvc.Send([]string{payload.EmailTo}, payload.Message, email.TplAddPolicyAddress); err != nil {
			http.Error(w, fmt.Sprintf("error sending email: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// AddPolicyCoverage handles the add-policy-coverage operation. It follows
// the same To/Message contract as the other handlers, additionally copying
// EmailCC recipients on the notification email.
func AddPolicyCoverage(emailsvc email.MailProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
			return
		}

		payload := AddPolicyCoverageReq{}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if err := emailsvc.SendWithCC([]string{payload.EmailTo}, payload.EmailCC, payload.Message, email.TplAddPolicyCoverage); err != nil {
			http.Error(w, fmt.Sprintf("error sending email: %v", err), http.StatusInternalServerError)
			return
		}
	}
}
