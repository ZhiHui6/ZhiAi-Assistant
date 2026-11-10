package win32

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shcore   = syscall.NewLazyDLL("shcore.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procUpdateLayeredWindow      = user32.NewProc("UpdateLayeredWindow")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procMonitorFromPoint         = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procEnumDisplayMonitors      = user32.NewProc("EnumDisplayMonitors")
	procGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	procGetDpiForSystem          = user32.NewProc("GetDpiForSystem")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procSetCursorPos             = user32.NewProc("SetCursorPos")
	procSendInput                = user32.NewProc("SendInput")
	procGetCurrentThreadID       = kernel32.NewProc("GetCurrentThreadId")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSetFocus                 = user32.NewProc("SetFocus")
	procGetActiveWindow          = user32.NewProc("GetActiveWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procRegisterHotKey           = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey         = user32.NewProc("UnregisterHotKey")
	procSetTimer                 = user32.NewProc("SetTimer")
	procKillTimer                = user32.NewProc("KillTimer")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procTrackPopupMenuEx         = user32.NewProc("TrackPopupMenuEx")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procRegisterWindowMessage    = user32.NewProc("RegisterWindowMessageW")
	procCreateIconFromRes        = user32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procGetDC                    = user32.NewProc("GetDC")
	procReleaseDC                = user32.NewProc("ReleaseDC")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procGetGuiResources          = user32.NewProc("GetGuiResources")
	procSetWindowRgn             = user32.NewProc("SetWindowRgn")
	procCreateRoundRectRgn       = gdi32.NewProc("CreateRoundRectRgn")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procMessageBeep              = user32.NewProc("MessageBeep")
	procPlaySoundW               = winmm.NewProc("PlaySoundW")

	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procSHGetFileInfoW  = shell32.NewProc("SHGetFileInfoW")
	procShellExecuteW   = shell32.NewProc("ShellExecuteW")

	procGetIconInfo        = user32.NewProc("GetIconInfo")
	procGetObjectW         = gdi32.NewProc("GetObjectW")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procCreateCompatDC     = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procGetWindowDC        = user32.NewProc("GetWindowDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procCreateBitmap       = gdi32.NewProc("CreateBitmap")
	procCreateIconIndirect = user32.NewProc("CreateIconIndirect")

	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	procGetModuleFileNameW = kernel32.NewProc("GetModuleFileNameW")

	procRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")

	procGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")
)

func errFalse(r1 uintptr) error {
	if r1 == 0 {
		if e := syscall.GetLastError(); e != nil && e != syscall.Errno(0) {
			return e
		}
		return syscall.EINVAL
	}
	return nil
}

func RegisterClassEx(wc *WNDCLASSEX) (uint16, error) {
	wc.CbSize = uint32(unsafe.Sizeof(*wc))
	r1, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if r1 == 0 {
		return 0, err
	}
	return uint16(r1), nil
}

func CreateWindowEx(exStyle uint32, className, windowName *uint16, style uint32, x, y, w, h int32, parent, menu, inst, param uintptr) uintptr {
	r1, _, _ := procCreateWindowExW.Call(
		uintptr(exStyle), uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)),
		uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, menu, inst, param)
	return r1
}

func DefWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r1, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r1
}

func DestroyWindow(hwnd uintptr) bool {
	r1, _, _ := procDestroyWindow.Call(hwnd)
	return r1 != 0
}

func GetMessage(msg *MSG, hwnd uintptr) int32 {
	r1, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(msg)), hwnd, 0, 0)
	return int32(r1)
}

func TranslateMessage(msg *MSG) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func DispatchMessage(msg *MSG) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func PostQuitMessage(code int32) {
	procPostQuitMessage.Call(uintptr(code))
}

func PostMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) bool {
	r1, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r1 != 0
}

func SetWindowPos(hwnd, insertAfter uintptr, x, y, cx, cy int32, flags uint32) bool {
	r1, _, _ := procSetWindowPos.Call(hwnd, insertAfter,
		uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(flags))
	return r1 != 0
}

func GetWindowRect(hwnd uintptr, r *RECT) bool {
	r1, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(r)))
	return r1 != 0
}

func GetCursorPos(p *POINT) bool {
	r1, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(p)))
	return r1 != 0
}

func MonitorFromPoint(p POINT, flags uint32) uintptr {
	if unsafe.Sizeof(uintptr(0)) == 8 {
		packed := uintptr(uint32(p.X)) | uintptr(uint32(p.Y))<<32
		r1, _, _ := procMonitorFromPoint.Call(packed, uintptr(flags))
		return r1
	}
	r1, _, _ := procMonitorFromPoint.Call(uintptr(uint32(p.X)), uintptr(uint32(p.Y)), uintptr(flags))
	return r1
}

func GetMonitorInfo(monitor uintptr, mi *MONITORINFO) bool {
	mi.CbSize = uint32(unsafe.Sizeof(*mi))
	r1, _, _ := procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(mi)))
	return r1 != 0
}

func EnumDisplayMonitors(callback uintptr, data uintptr) bool {
	r1, _, _ := procEnumDisplayMonitors.Call(0, 0, callback, data)
	return r1 != 0
}

func GetDpiForWindow(hwnd uintptr) uint32 {
	if err := procGetDpiForWindow.Find(); err != nil {
		return 96
	}
	r1, _, _ := procGetDpiForWindow.Call(hwnd)
	if r1 == 0 {
		return 96
	}
	return uint32(r1)
}

func GetDpiForMonitor(monitor uintptr) uint32 {
	if err := procGetDpiForMonitor.Find(); err != nil {
		return 96
	}
	var x, y uint32
	r1, _, _ := procGetDpiForMonitor.Call(monitor, 0, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	if r1 != 0 || x == 0 {
		return 96
	}
	return x
}

func SetWindowLongPtr(hwnd uintptr, index int32, value uintptr) uintptr {
	r1, _, _ := procSetWindowLongPtrW.Call(hwnd, uintptr(index), value)
	return r1
}

func GetWindowLongPtr(hwnd uintptr, index int32) uintptr {
	r1, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(index))
	return r1
}

func ShowWindow(hwnd uintptr, cmd int32) bool {
	r1, _, _ := procShowWindow.Call(hwnd, uintptr(cmd))
	return r1 != 0
}

func IsWindowVisible(hwnd uintptr) bool {
	r1, _, _ := procIsWindowVisible.Call(hwnd)
	return r1 != 0
}

func IsIconic(hwnd uintptr) bool {
	r1, _, _ := procIsIconic.Call(hwnd)
	return r1 != 0
}

func ClassName(hwnd uintptr) string {
	buf := make([]uint16, 128)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return UTF16ToString(buf[:n])
}

func MessageBeep(kind uint32) {
	procMessageBeep.Call(uintptr(kind))
}

func PlaySoundFile(path string) bool {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	const (
		sndAsync     = 0x0001
		sndNoDefault = 0x0002
		sndFilename  = 0x00020000
	)
	r1, _, _ := procPlaySoundW.Call(uintptr(unsafe.Pointer(ptr)), 0, sndFilename|sndAsync|sndNoDefault)
	return r1 != 0
}

func SetForegroundWindow(hwnd uintptr) bool {
	r1, _, _ := procSetForegroundWindow.Call(hwnd)
	return r1 != 0
}

func GetCurrentThreadID() uint32 {
	r1, _, _ := procGetCurrentThreadID.Call()
	return uint32(r1)
}

func GetWindowThreadProcessID(hwnd uintptr) uint32 {
	var pid uint32
	r1, _, _ := procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return uint32(r1)
}

func AttachThreadInput(attach, to uint32, enable bool) bool {
	flag := uintptr(0)
	if enable {
		flag = 1
	}
	r1, _, _ := procAttachThreadInput.Call(uintptr(attach), uintptr(to), flag)
	return r1 != 0
}

func BringWindowToTop(hwnd uintptr) bool {
	r1, _, _ := procBringWindowToTop.Call(hwnd)
	return r1 != 0
}

func SetFocus(hwnd uintptr) uintptr {
	r1, _, _ := procSetFocus.Call(hwnd)
	return r1
}

func GetActiveWindow() uintptr {
	r1, _, _ := procGetActiveWindow.Call()
	return r1
}

func SetCursorPos(x, y int32) bool {
	r1, _, _ := procSetCursorPos.Call(uintptr(x), uintptr(y))
	return r1 != 0
}

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type INPUT struct {
	Type uint32
	Pad  uint32
	Data [32]byte
}

func SendKeyInput(vk, scan uint16, flags uint32) bool {
	var in INPUT
	in.Type = 1
	ki := KEYBDINPUT{WVk: vk, WScan: scan, DwFlags: flags}
	copy(in.Data[:], unsafe.Slice((*byte)(unsafe.Pointer(&ki)), int(unsafe.Sizeof(ki))))
	r1, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	return r1 == 1
}

func SendMouseInput(flags uint32, dx, dy int32) bool {
	var in INPUT
	in.Type = 0
	mi := MOUSEINPUT{Dx: dx, Dy: dy, DwFlags: flags}
	copy(in.Data[:], unsafe.Slice((*byte)(unsafe.Pointer(&mi)), int(unsafe.Sizeof(mi))))
	r1, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	return r1 == 1
}

func GetForegroundWindow() uintptr {
	r1, _, _ := procGetForegroundWindow.Call()
	return r1
}

func FindWindow(className, windowName *uint16) uintptr {
	r1, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)))
	return r1
}

func RegisterHotKey(hwnd uintptr, id int32, mods, vk uint32) error {
	r1, _, err := procRegisterHotKey.Call(hwnd, uintptr(id), uintptr(mods), uintptr(vk))
	if r1 == 0 {
		if err == nil || err == syscall.Errno(0) {
			return syscall.EINVAL
		}
		return err
	}
	return nil
}

func UnregisterHotKey(hwnd uintptr, id int32) bool {
	r1, _, _ := procUnregisterHotKey.Call(hwnd, uintptr(id))
	return r1 != 0
}

func SetTimer(hwnd uintptr, id uintptr, elapse uint32, fn uintptr) uintptr {
	r1, _, _ := procSetTimer.Call(hwnd, id, uintptr(elapse), fn)
	return r1
}

func KillTimer(hwnd uintptr, id uintptr) bool {
	r1, _, _ := procKillTimer.Call(hwnd, id)
	return r1 != 0
}

func CreatePopupMenu() uintptr {
	r1, _, _ := procCreatePopupMenu.Call()
	return r1
}

func AppendMenu(menu uintptr, flags uint32, id uintptr, text *uint16) bool {
	r1, _, _ := procAppendMenuW.Call(menu, uintptr(flags), id, uintptr(unsafe.Pointer(text)))
	return r1 != 0
}

func TrackPopupMenuEx(menu uintptr, flags uint32, x, y int32, hwnd, params uintptr) int32 {
	r1, _, _ := procTrackPopupMenuEx.Call(menu, uintptr(flags), uintptr(x), uintptr(y), hwnd, params)
	return int32(r1)
}

func DestroyMenu(menu uintptr) bool {
	r1, _, _ := procDestroyMenu.Call(menu)
	return r1 != 0
}

func RegisterWindowMessage(name *uint16) uint32 {
	r1, _, _ := procRegisterWindowMessage.Call(uintptr(unsafe.Pointer(name)))
	return uint32(r1)
}

func ShellNotifyIcon(message uint32, nid *NOTIFYICONDATA) error {
	nid.CbSize = uint32(unsafe.Sizeof(*nid))
	r1, _, _ := procShellNotifyIcon.Call(uintptr(message), uintptr(unsafe.Pointer(nid)))
	if r1 == 0 {
		return syscall.EINVAL
	}
	return nil
}

func CreateIconFromResourceEx(bits []byte, size int32) uintptr {
	if len(bits) == 0 {
		return 0
	}
	r1, _, _ := procCreateIconFromRes.Call(
		uintptr(unsafe.Pointer(&bits[0])), uintptr(len(bits)), 1, 0x00030000,
		uintptr(size), uintptr(size), 0)
	return r1
}

func DestroyIcon(icon uintptr) bool {
	r1, _, _ := procDestroyIcon.Call(icon)
	return r1 != 0
}

func GetDC(hwnd uintptr) uintptr {
	r1, _, _ := procGetDC.Call(hwnd)
	return r1
}

func ReleaseDC(hwnd, dc uintptr) int32 {
	r1, _, _ := procReleaseDC.Call(hwnd, dc)
	return int32(r1)
}

func GetSystemMetrics(index int32) int32 {
	r1, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(r1)
}

func GetGuiResources(process uintptr, flags uint32) uint32 {
	r1, _, _ := procGetGuiResources.Call(process, uintptr(flags))
	return uint32(r1)
}

func SHGetFileInfo(path string, fileAttributes uint32, flags uint32) (SHFILEINFO, bool) {
	var info SHFILEINFO
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return info, false
	}
	r1, _, _ := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(p)), uintptr(fileAttributes),
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info), uintptr(flags))
	return info, r1 != 0
}

func GetIconInfo(icon uintptr) (ICONINFO, bool) {
	var ii ICONINFO
	r1, _, _ := procGetIconInfo.Call(icon, uintptr(unsafe.Pointer(&ii)))
	return ii, r1 != 0
}

func GetObject(handle uintptr, buf unsafe.Pointer, size int32) int32 {
	r1, _, _ := procGetObjectW.Call(handle, uintptr(size), uintptr(buf))
	return int32(r1)
}

func GetDIBits(dc, bitmap uintptr, start, lines uint32, bits unsafe.Pointer, bi *BITMAPINFOHEADER) int32 {
	r1, _, _ := procGetDIBits.Call(dc, bitmap, uintptr(start), uintptr(lines),
		uintptr(bits), uintptr(unsafe.Pointer(bi)), DIB_RGB_COLORS)
	return int32(r1)
}

func DeleteObject(h uintptr) bool {
	r1, _, _ := procDeleteObject.Call(h)
	return r1 != 0
}

func CreateCompatibleDC(dc uintptr) uintptr {
	r1, _, _ := procCreateCompatDC.Call(dc)
	return r1
}

func DeleteDC(dc uintptr) bool {
	r1, _, _ := procDeleteDC.Call(dc)
	return r1 != 0
}

func BitBlt(dst uintptr, x, y, w, h int32, src uintptr, sx, sy int32, rop uint32) bool {
	r1, _, _ := procBitBlt.Call(dst, uintptr(x), uintptr(y), uintptr(w), uintptr(h), src,
		uintptr(sx), uintptr(sy), uintptr(rop))
	return r1 != 0
}

func ShellExecute(hwnd uintptr, verb, file, params, dir string, show int32) error {
	op, _ := syscall.UTF16PtrFromString(verb)
	f, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	p, _ := syscall.UTF16PtrFromString(params)
	d, _ := syscall.UTF16PtrFromString(dir)
	r1, _, _ := procShellExecuteW.Call(hwnd,
		uintptr(unsafe.Pointer(op)), uintptr(unsafe.Pointer(f)),
		uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(d)), uintptr(show))
	if r1 <= 32 {
		return syscall.EINVAL
	}
	return nil
}

func GetModuleHandle() uintptr {
	r1, _, _ := procGetModuleHandleW.Call(0)
	return r1
}

func ModuleFileName() string {
	buf := make([]uint16, 1024)
	r1, _, _ := procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r1 == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:r1])
}

func SetWindowRgn(hwnd, rgn uintptr, redraw bool) int32 {
	redrawFlag := uintptr(0)
	if redraw {
		redrawFlag = 1
	}
	r1, _, _ := procSetWindowRgn.Call(hwnd, rgn, redrawFlag)
	return int32(r1)
}

func CreateRoundRectRgn(x1, y1, x2, y2, w, h int32) uintptr {
	r1, _, _ := procCreateRoundRectRgn.Call(uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2), uintptr(w), uintptr(h))
	return r1
}

func CreateDIBSection(dc uintptr, bi *BITMAPINFOHEADER, bits *unsafe.Pointer) uintptr {
	r1, _, _ := procCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(bi)), DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(bits)), 0, 0)
	return r1
}

func SelectObject(dc, obj uintptr) uintptr {
	r1, _, _ := procSelectObject.Call(dc, obj)
	return r1
}

func CreateBitmap(w, h int32, planes, bitsPerPixel uint32, bits unsafe.Pointer) uintptr {
	r1, _, _ := procCreateBitmap.Call(uintptr(w), uintptr(h), uintptr(planes), uintptr(bitsPerPixel), uintptr(bits))
	return r1
}

func CreateIconIndirect(ii *ICONINFO) uintptr {
	r1, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(ii)))
	return r1
}

func UpdateLayeredWindow(hwnd, hdcDst uintptr, dst *POINT, size *SIZE, hdcSrc uintptr, src *POINT, blend *BLENDFUNCTION, flags uint32) bool {
	r1, _, _ := procUpdateLayeredWindow.Call(hwnd, hdcDst, uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(size)), hdcSrc, uintptr(unsafe.Pointer(src)), 0,
		uintptr(unsafe.Pointer(blend)), uintptr(flags))
	return r1 != 0
}

func UTF16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

func UTF16ToString(buf []uint16) string {
	return syscall.UTF16ToString(buf)
}

func UTF16FromString(s string) []uint16 {
	out, _ := syscall.UTF16FromString(s)
	return out
}

func SendMouseWheel(delta int32) bool {
	var in INPUT
	in.Type = 0
	mi := MOUSEINPUT{MouseData: uint32(delta), DwFlags: 0x0800}
	copy(in.Data[:], unsafe.Slice((*byte)(unsafe.Pointer(&mi)), int(unsafe.Sizeof(mi))))
	r1, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	return r1 == 1
}

func UserDefaultUILanguage() uint16 {
	r, _, _ := kernel32.NewProc("GetUserDefaultUILanguage").Call()
	return uint16(r)
}

var (
	procFillRect              = user32.NewProc("FillRect")
	procDrawTextW             = user32.NewProc("DrawTextW")
	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procCreateFontIndirectW   = gdi32.NewProc("CreateFontIndirectW")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
)

func FillRect(hdc uintptr, r *RECT, brush uintptr) int32 {
	r1, _, _ := procFillRect.Call(hdc, uintptr(unsafe.Pointer(r)), brush)
	return int32(r1)
}

func DrawText(hdc uintptr, text *uint16, count int32, r *RECT, flags uint32) int32 {
	r1, _, _ := procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(text)), uintptr(count), uintptr(unsafe.Pointer(r)), uintptr(flags))
	return int32(r1)
}

func CreateSolidBrush(color uint32) uintptr {
	r1, _, _ := procCreateSolidBrush.Call(uintptr(color))
	return r1
}

func SetBkMode(hdc uintptr, mode int32) int32 {
	r1, _, _ := procSetBkMode.Call(hdc, uintptr(mode))
	return int32(r1)
}

func SetTextColor(hdc uintptr, color uint32) uint32 {
	r1, _, _ := procSetTextColor.Call(hdc, uintptr(color))
	return uint32(r1)
}

func CreateFontIndirectW(lf *LOGFONTW) uintptr {
	r1, _, _ := procCreateFontIndirectW.Call(uintptr(unsafe.Pointer(lf)))
	return r1
}

func TextExtent(hdc uintptr, text *uint16, count int32, size *SIZE) bool {
	r1, _, _ := procGetTextExtentPoint32W.Call(hdc, uintptr(unsafe.Pointer(text)), uintptr(count), uintptr(unsafe.Pointer(size)))
	return r1 != 0
}
