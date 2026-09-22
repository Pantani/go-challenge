package mockemail

import "github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"

// SendLog records a single delivery to a single recipient.
type SendLog struct {
	To   string
	Tpl  email.TplID
	Vars map[string]any
}

// SendLogs is an ordered list of recorded deliveries, oldest first.
type SendLogs []SendLog

// IsEmpty reports whether nothing has been recorded.
func (s SendLogs) IsEmpty() bool { return len(s) == 0 }

// Last returns the most recent log, or nil when nothing has been sent, so
// callers can inspect it without first guarding against an empty log.
func (s SendLogs) Last() *SendLog {
	if len(s) == 0 {
		return nil
	}
	return &s[len(s)-1]
}
