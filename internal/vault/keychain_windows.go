//go:build windows

package vault

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

// Windows Credential Manager implementation using Win32 API
type windowsKeychain struct{}

var (
	advapi32        = syscall.NewLazyDLL("advapi32.dll")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

type winCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

func (k *windowsKeychain) Available() bool {
	return runtime.GOOS == "windows"
}

func (k *windowsKeychain) Set(key string, value []byte) error {
	target := fmt.Sprintf("%s/%s", keychainService, key)
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("utf16 target: %w", err)
	}
	userPtr, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return fmt.Errorf("utf16 user: %w", err)
	}

	blob := value
	var blobPtr *byte
	if len(blob) > 0 {
		blobPtr = &blob[0]
	}

	blobSize := len(blob)
	if blobSize > 0xFFFFFFFF {
		return fmt.Errorf("credential too large (max 4GB supported by Windows API)")
	}

	cred := winCredential{
		Type:               credTypeGeneric,
		TargetName:         targetPtr,
		UserName:           userPtr,
		CredentialBlobSize: uint32(blobSize),
		CredentialBlob:     blobPtr,
		Persist:            credPersistLocalMachine,
	}

	r1, _, err := procCredWriteW.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if r1 == 0 {
		if err != nil {
			return err
		}
		return fmt.Errorf("CredWriteW failed with unknown error")
	}
	return nil
}

func (k *windowsKeychain) Get(key string) ([]byte, error) {
	target := fmt.Sprintf("%s/%s", keychainService, key)
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return nil, fmt.Errorf("utf16 target: %w", err)
	}

	var credPtr *winCredential
	r1, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetPtr)),
		credTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&credPtr)),
	)

	if r1 == 0 {
		return nil, err
	}
	if credPtr == nil {
		return nil, fmt.Errorf("CredReadW returned success but null pointer")
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(credPtr)))

	if credPtr.CredentialBlobSize == 0 || credPtr.CredentialBlob == nil {
		return []byte{}, nil
	}

	blob := make([]byte, credPtr.CredentialBlobSize)
	src := unsafe.Slice(credPtr.CredentialBlob, credPtr.CredentialBlobSize)
	copy(blob, src)

	return blob, nil
}

func (k *windowsKeychain) Delete(key string) error {
	target := fmt.Sprintf("%s/%s", keychainService, key)
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("utf16 target: %w", err)
	}

	r1, _, err := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetPtr)),
		credTypeGeneric,
		0,
	)

	if r1 == 0 {
		return err
	}
	return nil
}

func platformKeychain() KeychainStore {
	return &windowsKeychain{}
}
