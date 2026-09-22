package handlers

import (
	"encoding/json"
)

type AddPolicyVehicleReq struct {
	EmailTo string          `json:"email_to"`
	Message json.RawMessage `json:"message"`
}

type AddPolicyDriverReq struct {
	EmailTo string          `json:"email_to"`
	Message json.RawMessage `json:"message"`
}

type AddPolicyAddressReq struct {
	EmailTo string          `json:"email_to"`
	Message json.RawMessage `json:"message"`
}

// AddPolicyCoverageReq is the add-policy-coverage payload. It extends the
// shared To/Message contract with CC recipients, since policy coverage
// notifications need to copy additional parties.
type AddPolicyCoverageReq struct {
	EmailTo string          `json:"email_to"`
	EmailCC []string        `json:"email_cc"`
	Message json.RawMessage `json:"message"`
}
