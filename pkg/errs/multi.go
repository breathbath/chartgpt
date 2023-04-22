package errs

import (
	"github.com/pkg/errors"
	"strings"
)

type Multi struct {
	errors []error
}

func NewMulti() *Multi {
	return &Multi{
		errors: []error{},
	}
}

func (m *Multi) Add(err error) {
	if err == nil {
		return
	}

	m.errors = append(m.errors, err)
}

func (m *Multi) Err(format string, args ...interface{}) {
	var e error
	if len(args) == 0 {
		e = errors.New(format)
	} else {
		e = errors.Errorf(format, args...)
	}

	m.Add(e)
}

func (m *Multi) Error() string {
	if !m.HasErrors() {
		return ""
	}

	strErrs := make([]string, len(m.errors))

	for i := range m.errors {
		strErrs[i] = m.errors[i].Error()
	}

	return strings.Join(strErrs, "; ")
}

func (m *Multi) HasErrors() bool {
	return len(m.errors) > 0
}
