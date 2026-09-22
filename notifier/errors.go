package notifier

import "errors"

// Sentinel errors returned by Producer and its topic builders, so callers
// (and tests) can distinguish failure reasons with errors.Is instead of
// matching on message text.
var (
	ErrTopicNotRegistered  = errors.New("topic builder not registered")
	ErrMissingRecipients   = errors.New("notification requires recipient")
	ErrMissingTemplate     = errors.New("notification requires template")
	ErrMissingMailProvider = errors.New("notification requires mail provider")

	ErrInvalidDocumentUploadInput     = errors.New("invalid document upload input type")
	ErrDocumentUploadMissingRecipient = errors.New("document upload requires recipient")
	ErrDocumentUploadMissingDocument  = errors.New("document upload requires document")

	ErrInvalidOTPLoginInput      = errors.New("invalid otp login input type")
	ErrOTPLoginMissingRecipient  = errors.New("otp login requires recipient")
	ErrOTPLoginMissingCode       = errors.New("otp login requires otp code")
	ErrOTPLoginInvalidExpiration = errors.New("otp login requires valid expiration minutes")

	ErrInvalidPolicyRenewalInput        = errors.New("invalid policy renewal input type")
	ErrPolicyRenewalMissingRecipient    = errors.New("policy renewal requires recipient")
	ErrPolicyRenewalMissingPolicyNumber = errors.New("policy renewal requires policy number")
	ErrPolicyRenewalMissingRenewalDate  = errors.New("policy renewal requires renewal date")
)
