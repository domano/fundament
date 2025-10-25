//go:build darwin

package native

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/domano/fundament/internal/shimloader"
)

type SessionRef unsafe.Pointer

type Availability struct {
	State  int32
	Reason int32
}

type StreamCallback func(chunk string, final bool)

type cError struct {
	Code    int32
	_       int32
	Message *byte
}

type cBuffer struct {
	Data   *byte
	Length int64
}

type cAvailability struct {
	State  int32
	Reason int32
}

type cString struct {
	ptr *byte
	buf []byte
}

func newCString(s string) cString {
	b := append([]byte(s), 0)
	return cString{
		ptr: &b[0],
		buf: b,
	}
}

func (c cString) ptrOrNil() *byte {
	return c.ptr
}

type fundamentStreamCallback = uintptr

var (
	registerOnce sync.Once
	registerErr  error

	fnSessionCreate            func(*byte, *cError) SessionRef
	fnSessionDestroy           func(SessionRef)
	fnSessionRespond           func(SessionRef, *byte, *byte, *cBuffer, *cError) bool
	fnSessionRespondStructured func(SessionRef, *byte, *byte, *byte, *cBuffer, *cError) bool
	fnSessionStream            func(SessionRef, *byte, *byte, fundamentStreamCallback, unsafe.Pointer, *cError) bool
	fnSessionCheckAvailability func(*cAvailability, *cError) bool
	fnBufferFree               func(unsafe.Pointer)
	fnErrorFree                func(unsafe.Pointer)

	streamCallbackPtr fundamentStreamCallback
	streamHandles     sync.Map // map[unsafe.Pointer]*streamHandle
)

type streamHandle struct {
	callback StreamCallback
}

func init() {
	if err := shimloader.Initialize(); err != nil {
		panic(fmt.Sprintf("fundament: shim initialization failed: %v", err))
	}
	registerOnce.Do(func() {
		registerErr = registerFunctions()
	})
	if registerErr != nil {
		panic(fmt.Sprintf("fundament: shim symbol registration failed: %v", registerErr))
	}
}

func registerFunctions() error {
	if err := shimloader.Register("fundament_session_create", &fnSessionCreate); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_session_destroy", &fnSessionDestroy); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_session_respond", &fnSessionRespond); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_session_respond_structured", &fnSessionRespondStructured); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_session_stream", &fnSessionStream); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_session_check_availability", &fnSessionCheckAvailability); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_buffer_free", &fnBufferFree); err != nil {
		return err
	}
	if err := shimloader.Register("fundament_error_free", &fnErrorFree); err != nil {
		return err
	}

	streamCallbackPtr = purego.NewCallback(goFundamentStreamCallback)
	return nil
}

func SessionCreate(instructions string) (SessionRef, error) {
	cInstructions := newCString(instructions)

	var cerr cError
	ref := fnSessionCreate(cInstructions.ptrOrNil(), &cerr)
	if err := takeError(&cerr); err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, errors.New("fundament: session create failed without details")
	}
	return ref, nil
}

func SessionDestroy(ref SessionRef) {
	fnSessionDestroy(ref)
}

func SessionRespond(ref SessionRef, prompt string, optionsJSON string) (string, error) {
	cPrompt := newCString(prompt)
	cOptions := newCString(optionsJSON)

	var buf cBuffer
	var cerr cError
	ok := fnSessionRespond(ref, cPrompt.ptrOrNil(), cOptions.ptrOrNil(), &buf, &cerr)
	if err := takeError(&cerr); err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("fundament: respond failed without details")
	}
	if fnBufferFree != nil {
		defer fnBufferFree(unsafe.Pointer(&buf))
	}
	return fromBuffer(&buf), nil
}

func SessionRespondStructured(ref SessionRef, prompt, schemaJSON, optionsJSON string) (string, error) {
	cPrompt := newCString(prompt)
	cSchema := newCString(schemaJSON)
	cOptions := newCString(optionsJSON)

	var buf cBuffer
	var cerr cError
	ok := fnSessionRespondStructured(ref, cPrompt.ptrOrNil(), cSchema.ptrOrNil(), cOptions.ptrOrNil(), &buf, &cerr)
	if err := takeError(&cerr); err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("fundament: structured respond failed without details")
	}
	if fnBufferFree != nil {
		defer fnBufferFree(unsafe.Pointer(&buf))
	}
	return fromBuffer(&buf), nil
}

func SessionStream(ref SessionRef, prompt, optionsJSON string, cb StreamCallback) error {
	if cb == nil {
		return errors.New("fundament: stream callback must not be nil")
	}
	cPrompt := newCString(prompt)
	cOptions := newCString(optionsJSON)

	handlePtr := storeStreamCallback(cb)
	var cerr cError
	ok := fnSessionStream(ref, cPrompt.ptrOrNil(), cOptions.ptrOrNil(), streamCallbackPtr, handlePtr, &cerr)
	if err := takeError(&cerr); err != nil {
		releaseStreamCallback(handlePtr)
		return err
	}
	if !ok {
		releaseStreamCallback(handlePtr)
		return errors.New("fundament: streaming failed without details")
	}
	return nil
}

func CheckAvailability() (Availability, error) {
	var cav cAvailability
	var cerr cError
	ok := fnSessionCheckAvailability(&cav, &cerr)
	if err := takeError(&cerr); err != nil {
		return Availability{}, err
	}
	if !ok {
		return Availability{}, errors.New("fundament: availability check failed without details")
	}
	return Availability{
		State:  cav.State,
		Reason: cav.Reason,
	}, nil
}

func storeStreamCallback(cb StreamCallback) unsafe.Pointer {
	handle := &streamHandle{callback: cb}
	ptr := unsafe.Pointer(handle)
	streamHandles.Store(ptr, handle)
	return ptr
}

func releaseStreamCallback(ptr unsafe.Pointer) {
	streamHandles.Delete(ptr)
}

func goFundamentStreamCallback(chunk *byte, isFinal bool, userdata unsafe.Pointer) {
	value, ok := streamHandles.Load(userdata)
	if !ok {
		return
	}
	sh, _ := value.(*streamHandle)
	if sh == nil || sh.callback == nil {
		return
	}
	text := cStringValue(chunk)
	sh.callback(text, isFinal)
	if isFinal {
		releaseStreamCallback(userdata)
	}
}

func takeError(err *cError) error {
	if err == nil {
		return nil
	}
	if fnErrorFree != nil {
		defer fnErrorFree(unsafe.Pointer(err))
	}
	if err.Code == 0 && err.Message == nil {
		return nil
	}
	message := cStringValue(err.Message)
	if message == "" {
		message = "fundament: unknown error"
	}
	return errors.New(message)
}

func fromBuffer(buf *cBuffer) string {
	if buf == nil || buf.Data == nil || buf.Length <= 0 {
		return ""
	}
	length := int(buf.Length)
	data := unsafe.Slice(buf.Data, length)
	out := make([]byte, length)
	copy(out, data)
	return string(out)
}

func cStringValue(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	length := cStringLen(ptr)
	if length == 0 {
		return ""
	}
	data := unsafe.Slice(ptr, length)
	return string(data)
}

func cStringLen(ptr *byte) int {
	if ptr == nil {
		return 0
	}
	var n int
	for {
		b := *(*byte)(unsafe.Add(unsafe.Pointer(ptr), uintptr(n)))
		if b == 0 {
			break
		}
		n++
	}
	return n
}
