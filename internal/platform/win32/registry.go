package win32

import (
	"syscall"
	"unsafe"
)

const (
	HKEY_CURRENT_USER       = 0x80000001
	KEY_QUERY_VALUE         = 0x0001
	KEY_SET_VALUE           = 0x0002
	REG_SZ                  = 1
	REG_OPTION_NON_VOLATILE = 0
)

func RegCreateKey(hKey uintptr, subKey string) (uintptr, error) {
	p, err := syscall.UTF16PtrFromString(subKey)
	if err != nil {
		return 0, err
	}
	var handle uintptr
	var disposition uint32
	r1, _, _ := procRegCreateKeyExW.Call(hKey, uintptr(unsafe.Pointer(p)), 0, 0,
		REG_OPTION_NON_VOLATILE, KEY_SET_VALUE|KEY_QUERY_VALUE, 0,
		uintptr(unsafe.Pointer(&handle)), uintptr(unsafe.Pointer(&disposition)))
	if r1 != 0 {
		return 0, syscall.Errno(r1)
	}
	return handle, nil
}

func RegSetString(handle uintptr, name, value string) error {
	pn, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	pv, err := syscall.UTF16FromString(value)
	if err != nil {
		return err
	}
	r1, _, _ := procRegSetValueExW.Call(handle, uintptr(unsafe.Pointer(pn)), 0, REG_SZ,
		uintptr(unsafe.Pointer(&pv[0])), uintptr(len(pv)*2))
	if r1 != 0 {
		return syscall.Errno(r1)
	}
	return nil
}

func RegGetString(handle uintptr, name string) (string, bool) {
	pn, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return "", false
	}
	buf := make([]uint16, 1024)
	size := uint32(len(buf) * 2)
	var typ uint32
	r1, _, _ := procRegQueryValueExW.Call(handle, uintptr(unsafe.Pointer(pn)), 0,
		uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r1 != 0 {
		return "", false
	}
	n := int(size / 2)
	if n > 0 && buf[n-1] == 0 {
		n--
	}
	if n <= 0 || n > len(buf) {
		return "", false
	}
	return syscall.UTF16ToString(buf[:n]), true
}

func RegDeleteValue(handle uintptr, name string) error {
	pn, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	r1, _, _ := procRegDeleteValueW.Call(handle, uintptr(unsafe.Pointer(pn)))
	if r1 != 0 {
		return syscall.Errno(r1)
	}
	return nil
}

func RegCloseKey(handle uintptr) {
	procRegCloseKey.Call(handle)
}
