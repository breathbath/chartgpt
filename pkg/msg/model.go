package msg

type Sender struct {
	ID        string
	FirstName string
	LastName  string
}

func (s *Sender) GetID() string {
	if s == nil {
		return ""
	}

	return s.ID
}

type Request struct {
	Source  string
	ID      string
	Sender  *Sender
	Message string
	Meta    map[string]interface{}
}

type Type uint

const (
	Undefined Type = iota
	Success
	Error
)

type Response struct {
	Message string
	Type    Type
	Meta    map[string]interface{}
}
