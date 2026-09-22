package notifier

import (
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// TopicOTPLogin is the notification topic for one-time login codes.
const TopicOTPLogin = "otp-login"

// OTPLoginInput is the typed input for the otp-login topic. Recipient and
// OTPCode are required; ExpirationMins must be positive.
type OTPLoginInput struct {
	Recipient      string
	OTPCode        string
	ExpirationMins int
}

var otpLoginTopicBuilder = topicBuilder[OTPLoginInput]{
	topic:           TopicOTPLogin,
	tpl:             email.TplOTPLogin,
	errInvalidInput: ErrInvalidOTPLoginInput,
	build:           buildOTPLogin,
}

func buildOTPLogin(in OTPLoginInput) (string, map[string]any, error) {
	if strings.TrimSpace(in.Recipient) == "" {
		return "", nil, ErrOTPLoginMissingRecipient
	}
	if strings.TrimSpace(in.OTPCode) == "" {
		return "", nil, ErrOTPLoginMissingCode
	}
	if in.ExpirationMins <= 0 {
		return "", nil, ErrOTPLoginInvalidExpiration
	}
	// expirationMins is validated above, so the template can tell the user
	// how long the code stays valid.
	return in.Recipient, map[string]any{
		"otpCode":        in.OTPCode,
		"expirationMins": in.ExpirationMins,
	}, nil
}
