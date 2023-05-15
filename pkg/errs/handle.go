package errs

import (
	"fmt"

	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func Handle(err error, stop bool) {
	if err == nil {
		return
	}

	var errWithStack stackTracer
	ok := errors.As(err, &errWithStack)
	if !ok {
		if stop {
			panic(err)
		} else {
			logging.Error(err)
		}
		return
	}

	st := errWithStack.StackTrace()
	if stop {
		panic(fmt.Sprintf("%v\n%+v", err, st))
	} else {
		logging.Errorf("%v\n%+v", err, st)
	}
}
