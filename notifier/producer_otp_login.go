package notifier

import (
	"context"
	"fmt"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

const TopicOTPLogin = "otp-login"

type OTPLoginInput struct {
	Recipient      string
	OTPCode        string
	ExpirationMins int
}

type otpLoginTopicBuilder struct{}

func (otpLoginTopicBuilder) Topic() string { return TopicOTPLogin }

func (otpLoginTopicBuilder) BuildRequest(_ context.Context, input any) (Request, error) {

	typedInput, ok := input.(OTPLoginInput)
	if !ok {
		return Request{}, fmt.Errorf("invalid otp login input type")
	}
	if typedInput.Recipient == "" {
		return Request{}, fmt.Errorf("otp login requires recipient")
	}
	if typedInput.OTPCode == "" {
		return Request{}, fmt.Errorf("otp login requires otp code")
	}
	if typedInput.ExpirationMins <= 0 {
		return Request{}, fmt.Errorf("otp login requires valid expiration minutes")
	}
	return Request{
		Topic:      TopicOTPLogin,
		Recipients: []string{typedInput.Recipient},
		Template:   email.TplOTPLogin,
		Vars:       map[string]any{"otpCode": typedInput.OTPCode},
	}, nil
}
