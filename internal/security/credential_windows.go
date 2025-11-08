//go:build windows
// +build windows

package security

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	advapi32                 = syscall.NewLazyDLL("advapi32.dll")
	credWriteW               = advapi32.NewProc("CredWriteW")
	credReadW                = advapi32.NewProc("CredReadW")
	credDeleteW              = advapi32.NewProc("CredDeleteW")
	credFree                 = advapi32.NewProc("CredFree")
)

const (
	CRED_TYPE_GENERIC         = 1
	CRED_PERSIST_LOCAL_MACHINE = 2
)

type credential struct {
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

// storeInCredentialManager stores the token in Windows Credential Manager
func (te *TokenEncryptor) storeInCredentialManager(token string) error {
	targetName, err := syscall.UTF16PtrFromString(credentialName)
	if err != nil {
		return fmt.Errorf("failed to convert target name: %w", err)
	}

	userName, err := syscall.UTF16PtrFromString("DeeMusic")
	if err != nil {
		return fmt.Errorf("failed to convert username: %w", err)
	}

	tokenBytes := []byte(token)
	
	cred := &credential{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(tokenBytes)),
		CredentialBlob:     &tokenBytes[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
		UserName:           userName,
	}

	ret, _, err := credWriteW.Call(
		uintptr(unsafe.Pointer(cred)),
		0,
	)

	if ret == 0 {
		return fmt.Errorf("CredWriteW failed: %w", err)
	}

	return nil
}

// retrieveFromCredentialManager retrieves the token from Windows Credential Manager
func (te *TokenEncryptor) retrieveFromCredentialManager() (string, error) {
	targetName, err := syscall.UTF16PtrFromString(credentialName)
	if err != nil {
		return "", fmt.Errorf("failed to convert target name: %w", err)
	}

	var cred *credential
	ret, _, err := credReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0,
		uintptr(unsafe.Pointer(&cred)),
	)

	if ret == 0 {
		return "", fmt.Errorf("CredReadW failed: %w", err)
	}

	defer credFree.Call(uintptr(unsafe.Pointer(cred)))

	// Extract token from credential blob
	tokenBytes := make([]byte, cred.CredentialBlobSize)
	for i := uint32(0); i < cred.CredentialBlobSize; i++ {
		tokenBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(cred.CredentialBlob)) + uintptr(i)))
	}

	return string(tokenBytes), nil
}

// deleteFromCredentialManager deletes the token from Windows Credential Manager
func (te *TokenEncryptor) deleteFromCredentialManager() error {
	targetName, err := syscall.UTF16PtrFromString(credentialName)
	if err != nil {
		return fmt.Errorf("failed to convert target name: %w", err)
	}

	ret, _, err := credDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0,
	)

	if ret == 0 {
		return fmt.Errorf("CredDeleteW failed: %w", err)
	}

	return nil
}
