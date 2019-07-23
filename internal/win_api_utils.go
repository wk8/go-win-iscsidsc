package internal

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
)

var (
	// TODO: we could (should?) check the version
	iscsidscDLL = windows.NewLazySystemDLL(getEnv("GO_WIN_ISCSI_DLL_NAME", "iscsidsc.dll"))

	// InitialAPIBufferSize is the size of the buffer used for the 1st call to APIs that need one.
	// It should big enough to ensure we won't need to make another call with a bigger buffer in most situations.
	// Having it as a var and not a constant allows overriding it during tests.
	// Note that on some versions of Windows, if this is too big, some API calls might result in ERROR_NOACCESS
	// errors (...?)
	InitialAPIBufferSize uintptr = 100000
)

// GetDllProc returns a handle to a proc from the system's iscsidsc.dll.
func GetDllProc(name string) *windows.LazyProc {
	return iscsidscDLL.NewProc(getEnv("GO_WIN_ISCSI_DLL_PROCS_PREFIX", "") + name)
}

//go:uintptrescapes
//go:noinline

// CallWinAPI makes a call to Windows' API.
func CallWinAPI(proc *windows.LazyProc, args ...uintptr) (uintptr, error) {
	if err := proc.Find(); err != nil {
		return 0, errors.Wrapf(err, "Unable to locate %q function in DLL %q", proc.Name, iscsidscDLL.Name)
	}

	exitCode, _, _ := proc.Call(args...)

	if exitCode == 0 {
		return exitCode, nil
	}
	return exitCode, iscsidsc.NewWinAPICallError(proc.Name, exitCode)
}

// HandleBufferedWinAPICall is a helper for Windows API calls listing objects, that always follow the same pattern:
// the caller has to allocate a buffer, and the proc fills that buffer, returning an object count and a byte count.
// typeSize is the size, in bytes, of the type the API calls expect the buffer to be (eg 1 for CHAR, 2 for WCHAR, etc...)
func HandleBufferedWinAPICall(f func(s, c, b uintptr) (uintptr, error), procName string, typeSize uintptr) (buffer []byte, bufferPointer uintptr, count int32, err error) {
	bufferSize := InitialAPIBufferSize/typeSize + 1
	var exitCode uintptr

	for {
		buffer = make([]byte, bufferSize*typeSize)

		exitCode, bufferPointer, err = makeBufferedWinAPICall(
			f,
			uintptr(unsafe.Pointer(&bufferSize)),
			uintptr(unsafe.Pointer(&count)),
			uintptr(unsafe.Pointer(&buffer[0])),
		)

		if exitCode != uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {
			if exitCode == 0 {
				// sanity check: the reported size should be smaller than the expected size
				if bufferSize*typeSize <= uintptr(len(buffer)) {
					buffer = buffer[:bufferSize*typeSize]
				} else {
					err = errors.Errorf("Call to %q successful, but reported buffer size %d bigger than actual size %d", procName, bufferSize*typeSize, len(buffer))
				}
			}

			return
		}

		// sanity check: is the new buffer size indeed bigger than the previous one?
		if bufferSize*typeSize <= uintptr(len(buffer)) {
			// this should never happen
			err = errors.Errorf("Error when calling %q: buffer of size %d deemed too small but bigger than the new advised size of %d", procName, len(buffer), bufferSize*typeSize)
			return
		}
		// try again with a bigger buffer
	}
}

//go:uintptrescapes
//go:noinline

// ensures the none of the arguments will be moved by the GC before we return; in particular,
// allows saving the position of the buffer in memory when passed to the Win API proc.
func makeBufferedWinAPICall(f func(s, c, b uintptr) (uintptr, error), size, count, buffer uintptr) (uintptr, uintptr, error) {
	exitCode, err := f(size, count, buffer)
	return exitCode, buffer, err
}

// BoolToByte converts a boolean to a C++ byte.
func BoolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// getEnv looks up the `key` env variable, or returns `ifAbsent` if it's not defined
// or empty.
func getEnv(key, ifAbsent string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return ifAbsent
}
