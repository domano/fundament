//go:build darwin && cgo

package native

/*
#cgo CFLAGS: -I${SRCDIR}/../../include
#cgo LDFLAGS: -L${SRCDIR}/../../swift/FundamentShim/.build/Release -Wl,-rpath,${SRCDIR}/../../swift/FundamentShim/.build/Release -lFundamentShim -framework Foundation -framework FoundationModels
#include "fundament.h"
#include <stdlib.h>
extern void goFundamentStreamCallback(char *chunk, _Bool final, void *userdata);
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

type SessionRef unsafe.Pointer

type Availability struct {
	State  int32
	Reason int32
}

type StreamCallback func(chunk string, final bool)

func SessionCreate(instructions string) (SessionRef, error) {
	cInstructions := toCString(instructions)
	defer freeCString(cInstructions)

	var cErr C.fundament_error
	ref := C.fundament_session_create(cInstructions, &cErr)
	err := takeError(&cErr)
	if ref == nil {
		if err == nil {
			err = errors.New("fundament: session create failed without details")
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return SessionRef(ref), nil
}

func SessionDestroy(ref SessionRef) {
	C.fundament_session_destroy(C.fundament_session_ref(ref))
}

func SessionRespond(ref SessionRef, prompt string, optionsJSON string) (string, error) {
	cPrompt := toCString(prompt)
	defer freeCString(cPrompt)
	cOptions := toCString(optionsJSON)
	defer freeCString(cOptions)

	var cBuffer C.fundament_buffer
	var cErr C.fundament_error
	ok := C.fundament_session_respond(C.fundament_session_ref(ref), cPrompt, cOptions, &cBuffer, &cErr)
	err := takeError(&cErr)
	if !ok {
		if err != nil {
			return "", err
		}
		return "", errors.New("fundament: respond failed without details")
	}
	if err != nil {
		return "", err
	}
	defer C.fundament_buffer_free(unsafe.Pointer(&cBuffer))
	return fromBuffer(cBuffer), nil
}

func SessionRespondStructured(ref SessionRef, prompt, schemaJSON, optionsJSON string) (string, error) {
	cPrompt := toCString(prompt)
	defer freeCString(cPrompt)
	cSchema := toCString(schemaJSON)
	defer freeCString(cSchema)
	cOptions := toCString(optionsJSON)
	defer freeCString(cOptions)

	var cBuffer C.fundament_buffer
	var cErr C.fundament_error
	ok := C.fundament_session_respond_structured(C.fundament_session_ref(ref), cPrompt, cSchema, cOptions, &cBuffer, &cErr)
	err := takeError(&cErr)
	if !ok {
		if err != nil {
			return "", err
		}
		return "", errors.New("fundament: structured respond failed without details")
	}
	if err != nil {
		return "", err
	}
	defer C.fundament_buffer_free(unsafe.Pointer(&cBuffer))
	return fromBuffer(cBuffer), nil
}

func SessionStream(ref SessionRef, prompt, optionsJSON string, cb StreamCallback) error {
	cPrompt := toCString(prompt)
	defer freeCString(cPrompt)
	cOptions := toCString(optionsJSON)
	defer freeCString(cOptions)

	handle := registerStreamCallback(cb)
	defer releaseStreamCallback(handle)

	var cErr C.fundament_error
	ok := C.fundament_session_stream(C.fundament_session_ref(ref), cPrompt, cOptions, C.fundament_stream_cb(C.goFundamentStreamCallback), unsafe.Pointer(handle), &cErr)
	err := takeError(&cErr)
	if !ok {
		if err != nil {
			return err
		}
		return errors.New("fundament: streaming failed without details")
	}
	return err
}

func CheckAvailability() (Availability, error) {
	var cAvailability C.fundament_availability
	var cErr C.fundament_error
	ok := C.fundament_session_check_availability(&cAvailability, &cErr)
	err := takeError(&cErr)
	if !ok {
		if err != nil {
			return Availability{}, err
		}
		return Availability{}, errors.New("fundament: availability check failed without details")
	}
	if err != nil {
		return Availability{}, err
	}
	return Availability{
		State:  int32(cAvailability.state),
		Reason: int32(cAvailability.reason),
	}, nil
}

// Helpers

func toCString(v string) *C.char {
	if v == "" {
		return C.CString("")
	}
	return C.CString(v)
}

func freeCString(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

func fromBuffer(buffer C.fundament_buffer) string {
	if buffer.data == nil || buffer.length == 0 {
		return ""
	}
	return C.GoStringN(buffer.data, C.int(buffer.length))
}

func takeError(err *C.fundament_error) error {
	if err == nil {
		return nil
	}
	defer C.fundament_error_free(unsafe.Pointer(err))
	if err.code == 0 && err.message == nil {
		return nil
	}
	message := ""
	if err.message != nil {
		message = C.GoString(err.message)
	}
	if message == "" {
		message = "fundament: unknown error"
	}
	return errors.New(message)
}

//export goFundamentStreamCallback
func goFundamentStreamCallback(cChunk *C.char, final C.bool, userData unsafe.Pointer) {
	handle := (*streamHandle)(userData)
	if handle == nil {
		return
	}
	chunk := ""
	if cChunk != nil {
		chunk = C.GoString(cChunk)
	}
	handle.invoke(chunk, bool(final))
}

type streamHandle struct {
	callback StreamCallback
}

var (
	streamCallbacks sync.Map
)

func registerStreamCallback(cb StreamCallback) *streamHandle {
	handle := &streamHandle{
		callback: cb,
	}
	streamCallbacks.Store(handle, struct{}{})
	return handle
}

func releaseStreamCallback(handle *streamHandle) {
	if handle == nil {
		return
	}
	streamCallbacks.Delete(handle)
}

func (h *streamHandle) invoke(chunk string, final bool) {
	if h.callback != nil {
		h.callback(chunk, final)
	}
	if final {
		releaseStreamCallback(h)
	}
}
