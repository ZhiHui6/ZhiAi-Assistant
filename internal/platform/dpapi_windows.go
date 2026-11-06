package platform

import (
	"errors"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32                 = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData    = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData  = crypt32.NewProc("CryptUnprotectData")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procLocalFree           = kernel32.NewProc("LocalFree")
	cryptProtectUIForbidden = uint32(1)
)

func ProtectSecret(data []byte) ([]byte, error) {
	return cryptData(data, procCryptProtectData)
}

func UnprotectSecret(data []byte) ([]byte, error) {
	return cryptData(data, procCryptUnprotectData)
}

func cryptData(data []byte, proc *syscall.LazyProc) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("空数据")
	}
	in := dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
	var out dataBlob
	r1, _, err := proc.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectUIForbidden),
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		if err != nil && err != syscall.Errno(0) {
			return nil, err
		}
		return nil, errors.New("DPAPI 操作失败")
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	result := make([]byte, int(out.cbData))
	copy(result, unsafe.Slice(out.pbData, int(out.cbData)))
	return result, nil
}
