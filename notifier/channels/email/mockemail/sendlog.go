package mockemail

import "github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"

type SendLog struct {
	To   string
	Tpl  email.TplID
	Vars map[string]any
}

type SendLogs []SendLog

func (s SendLogs) IsEmpty() bool { return len(s) == 0 }

func (s SendLogs) Last() *SendLog { return &s[len(s)-1] }
