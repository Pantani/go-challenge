package mockemail

import (
	"encoding/json"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

// SendLog records a single recipient delivery made through Client.
type SendLog struct {
	to      string
	cc      []string
	tplID   email.TplID
	message json.RawMessage
}

// ExtractTo returns the primary recipient of this send.
func (sl *SendLog) ExtractTo() string {
	return sl.to
}

// ExtractCC returns the recipients copied on this send, if any.
func (sl *SendLog) ExtractCC() []string {
	return sl.cc
}

// ExtractTplID returns the template used for this send.
func (sl *SendLog) ExtractTplID() email.TplID {
	return sl.tplID
}

// ExtractMessage returns the raw message payload for this send.
func (sl *SendLog) ExtractMessage() json.RawMessage {
	return sl.message
}

// SendLogs is a chronological list of recorded sends.
type SendLogs []SendLog

// IsEmpty reports whether no sends have been recorded.
func (s SendLogs) IsEmpty() bool {
	return len(s) == 0
}

// Last will return the last created log
func (s SendLogs) Last() *SendLog {
	return &s[len(s)-1]
}
