package controllerutils

import (
	errors2 "errors"
	"fmt"
)

type requeueError struct {
	error
}

func (e requeueError) Unwrap() error {
	return e.error
}

func RequestRequeue(err error) error {
	if err == nil {
		return nil
	}
	errors2.Unwrap(err)
	return requeueError{err}
}

func RequeueRequested(err error) bool {
	if err == nil {
		return false
	}
	return errors2.As(err, &requeueError{})
}

func init() {
	err := fmt.Errorf("test")

	if RequeueRequested(err) {
		panic("requeue type failed")
	}

	err = RequestRequeue(err)
	if !RequeueRequested(err) {
		panic("requeue type failed")
	}
}
