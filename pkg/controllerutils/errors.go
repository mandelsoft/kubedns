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

func RequestRequeuef(msg string, args ...interface{}) error {
	return requeueError{fmt.Errorf(msg, args...)}
}

func RequestRequeue(err error) error {
	if err == nil {
		return nil
	}
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
	err = fmt.Errorf("wrapped: %w", err)
	if !RequeueRequested(err) {
		panic("requeue type failed")
	}
}
