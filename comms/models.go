package comms

import (
	"encoding/json"
)

// AddPolicyVehicleReq is the add-policy-vehicle payload.
type AddPolicyVehicleReq struct {
	EmailTo string          `json:"email_to"`
	Message json.RawMessage `json:"message"`
}

// AddPolicyDriverReq is the add-policy-driver payload.
type AddPolicyDriverReq struct {
	EmailTo string          `json:"email_to"`
	Message json.RawMessage `json:"message"`
}

// AddPolicyAddressReq is the add-policy-address payload.
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
