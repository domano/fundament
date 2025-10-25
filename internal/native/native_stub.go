//go:build !darwin

package native

import (
	"errors"
	"unsafe"
)

type SessionRef unsafe.Pointer

type Availability struct {
	State  int32
	Reason int32
}

type StreamCallback func(chunk string, final bool)

func SessionCreate(string) (SessionRef, error) {
	return nil, errors.New("fundament: macOS 26 is required")
}

func SessionDestroy(SessionRef) {}

func SessionRespond(SessionRef, string, string) (string, error) {
	return "", errors.New("fundament: macOS 26 is required")
}

func SessionRespondStructured(SessionRef, string, string, string) (string, error) {
	return "", errors.New("fundament: macOS 26 is required")
}

func SessionStream(SessionRef, string, string, StreamCallback) error {
	return errors.New("fundament: macOS 26 is required")
}

func CheckAvailability() (Availability, error) {
	return Availability{}, errors.New("fundament: macOS 26 is required")
}
