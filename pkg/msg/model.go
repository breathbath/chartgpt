package msg

import (
	"fmt"
	"io"
	"strings"
)

type Sender struct {
	ID,
	FirstName,
	UserName,
	LastName,
	Language string
}

func (s *Sender) GetID() string {
	if s == nil {
		return ""
	}

	return s.UserName
}

func (s *Sender) String() string {
	userFields := []string{}
	if s.FirstName != "" {
		userFields = append(userFields, "Имя: "+s.FirstName)
	}
	if s.LastName != "" {
		userFields = append(userFields, "Фамилия: "+s.LastName)
	}

	if len(userFields) == 0 {
		return ""
	}

	return strings.Join(userFields, ", ")
}

const FormatVoice = "voice"

type File struct {
	FileID     string
	UniqueID   string
	FileSize   int64
	FilePath   string
	FileLocal  string
	FileURL    string
	FileReader io.ReadCloser
	Format     string
}

func (f File) String() string {
	return fmt.Sprintf(
		"ID: %s, Size: %d, Path: %s, Local: %s, URL: %s, Format: %s",
		f.FileID, f.FileSize, f.FilePath, f.FileLocal, f.FileURL, f.Format,
	)
}

type Request struct {
	Platform string
	ID       string
	Sender   *Sender
	Message  string
	Meta     map[string]interface{}
	File     File
}

func (r Request) GetConversationID() string {
	conversationIDI, ok := r.Meta["conversation_id"]
	conversationID := ""
	if ok {
		conversationID = fmt.Sprint(conversationIDI)
	}

	return fmt.Sprintf("%s/%s/%s", strings.ToLower(r.Platform), conversationID, strings.ToLower(r.Sender.GetID()))
}

type Type uint

const (
	Undefined Type = iota
	Success
	Error
)

type ResponseMessage struct {
	Message        string
	Type           Type
	Options        *Options
	DelayedOptions *DelayedOptions
	Media          *Media
}

type Response struct {
	Messages []ResponseMessage
}

const MediaTypeImage = "image"
const MediaPathTypeUrl = "url"
const MediaPathTypeFile = "file"

type Media struct {
	Path            string
	Type            string
	PathType        string
	IsBeforeMessage bool
}
