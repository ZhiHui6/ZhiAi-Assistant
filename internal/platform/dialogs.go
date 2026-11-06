package platform

import (
	"errors"
	"path/filepath"
	"syscall"
	"unsafe"
	"zhi-ai-assistant/internal/i18n"

	"zhi-ai-assistant/internal/platform/win32"
)

var (
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	shell32x = syscall.NewLazyDLL("shell32.dll")

	procGetOpenFileNameW    = comdlg32.NewProc("GetOpenFileNameW")
	procSHBrowseForFolder   = shell32x.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDList = shell32x.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree       = ole32.NewProc("CoTaskMemFree")
	procCoInitializeEx      = ole32.NewProc("CoInitializeEx")
)

type openFileName struct {
	StructSize    uint32
	HwndOwner     uintptr
	Instance      uintptr
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustFilter uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxFileTitle  uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExt        *uint16
	CustData      uintptr
	Hook          uintptr
	TemplateName  *uint16
	ReservedPtr   uintptr
	Reserved      uint32
	FlagsEx       uint32
}

type browseInfo struct {
	HwndOwner   uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Callback    uintptr
	Param       uintptr
	Image       int32
}

const (
	ofnReadOnly        = 0x00000001
	ofnOverwritePrompt = 0x00000002
	ofnNoChangeDir     = 0x00000008
	ofnAllowMulti      = 0x00000200
	ofnPathMustExist   = 0x00000800
	ofnFileMustExist   = 0x00001000
	ofnHideReadOnly    = 0x00000004
	ofnExplorer        = 0x00080000

	bifReturnOnlyFSDirs = 0x0001
	bifNewDialogStyle   = 0x0040
)

func coInit() {
	if err := procCoInitializeEx.Find(); err == nil {
		procCoInitializeEx.Call(0, 2)
	}
}

func FilterString(pairs ...string) []uint16 {
	var out []uint16
	for _, p := range pairs {
		seg := win32.UTF16FromString(p)
		if len(seg) > 0 {
			out = append(out, seg[:len(seg)-1]...)
		}
		out = append(out, 0)
	}
	out = append(out, 0)
	return out
}

func OpenFileDialog(title string, multi bool) ([]string, error) {
	coInit()
	buf := make([]uint16, 32768)
	filter := FilterString(i18n.T("所有文件"), "*.*", i18n.T("提示音"), "*.wav", i18n.T("程序与快捷方式"), "*.exe;*.lnk;*.bat;*.cmd;*.msi", i18n.T("文件夹快捷方式"), "*.lnk")
	titlePtr := win32.UTF16(title)

	flags := uint32(ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir | ofnHideReadOnly)
	if multi {
		flags |= ofnAllowMulti
	}
	ofn := openFileName{
		StructSize: uint32(unsafe.Sizeof(openFileName{})),
		Filter:     &filter[0],
		File:       &buf[0],
		MaxFile:    uint32(len(buf)),
		Title:      titlePtr,
		Flags:      flags,
	}
	r1, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r1 == 0 {
		return nil, errors.New("已取消")
	}

	first := win32.UTF16ToString(buf)
	if !multi || !containsZeroPair(buf) {
		if first == "" {
			return nil, errors.New("未选择文件")
		}
		return []string{first}, nil
	}
	dir := first
	var out []string
	i := len(dir) + 1
	for i < len(buf) {
		if buf[i] == 0 {
			break
		}
		end := i
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		name := win32.UTF16ToString(buf[i:end])
		if name != "" {
			out = append(out, filepath.Join(dir, name))
		}
		i = end + 1
	}
	if len(out) == 0 {
		return []string{dir}, nil
	}
	return out, nil
}

func containsZeroPair(buf []uint16) bool {
	for i := 0; i+1 < len(buf); i++ {
		if buf[i] == 0 && buf[i+1] == 0 {
			return true
		}
	}
	return false
}

func BrowseFolder(title string) (string, error) {
	coInit()
	display := make([]uint16, 260)
	bi := browseInfo{
		DisplayName: &display[0],
		Title:       win32.UTF16(title),
		Flags:       bifReturnOnlyFSDirs | bifNewDialogStyle,
	}
	r1, _, _ := procSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if r1 == 0 {
		return "", errors.New("已取消")
	}
	defer procCoTaskMemFree.Call(r1)
	path := make([]uint16, 1024)
	ok, _, _ := procSHGetPathFromIDList.Call(r1, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return "", errors.New("无法解析所选目录")
	}
	p := win32.UTF16ToString(path)
	if p == "" {
		return "", errors.New("未选择目录")
	}
	return p, nil
}
