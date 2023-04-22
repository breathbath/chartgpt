package errs

import (
	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
)

func Handle(err error, stop bool) {
	if err == nil {
		return
	}

	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	checkErr, ok := errors.Cause(err).(stackTracer)
	if !ok {
		if stop {
			logging.Panic(err)
		} else {
			logging.Error(err)
		}
		return
	}

	st := checkErr.StackTrace()
	if stop {
		logging.Panicf("%v\n%+v", err, st)
	} else {
		logging.Errorf("%v\n%+v", err, st)
	}
}
