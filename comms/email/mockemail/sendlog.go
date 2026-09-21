package mockemail

import (
	"encoding/json"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

type SendLog struct {
	to      string
	tplID   email.TplID
	message json.RawMessage
}

func (sl *SendLog) ExtractTo() string {
	return sl.to
}

func (sl *SendLog) ExtractTplID() email.TplID {
	return sl.tplID
}

func (sl *SendLog) ExtractMessage() json.RawMessage {
	return sl.message
}

type SendLogs []SendLog

func (s SendLogs) IsEmpty() bool {
	return len(s) == 0
}

// Last will return the last created log
func (s SendLogs) Last() *SendLog {
	return &s[len(s)-1]
}
