package mockemail

import (
	"encoding/json"
	"slices"

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

// ExtractCC returns a copy of the recipients copied on this send, if any.
func (sl *SendLog) ExtractCC() []string {
	return slices.Clone(sl.cc)
}

// ExtractTplID returns the template used for this send.
func (sl *SendLog) ExtractTplID() email.TplID {
	return sl.tplID
}

// ExtractMessage returns a copy of the raw message payload for this send.
func (sl *SendLog) ExtractMessage() json.RawMessage {
	return slices.Clone(sl.message)
}

// SendLogs is a chronological list of recorded sends.
type SendLogs []SendLog

// IsEmpty reports whether no sends have been recorded.
func (s SendLogs) IsEmpty() bool {
	return len(s) == 0
}

// Last returns the most recently recorded log, or nil when none exist.
func (s SendLogs) Last() *SendLog {
	if len(s) == 0 {
		return nil
	}
	return &s[len(s)-1]
}
