package platform

import (
	"bytes"
	_ "embed"
	"errors"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
	"zhi-ai-assistant/internal/i18n"

	"zhi-ai-assistant/internal/buildinfo"
	"zhi-ai-assistant/internal/platform/win32"
)

//go:embed tray.png
var trayIconPNG []byte

const (
	timerCursor  = 1
	timerAnim    = 2
	hotkeyIDMain = 0x5301
	kickMessage  = win32.WM_APP + 3
	hotkeyMsg    = win32.WM_APP + 4
	settingsMsg  = win32.WM_APP + 5
	trayMsg      = win32.WM_APP + 6
	startupMsg   = win32.WM_APP + 7
	trayClickMsg = win32.WM_APP + 8
	trayOpenMsg  = win32.WM_APP + 9

	trayDedupe = 350 * time.Millisecond

	hostClassName = "ZhiAi-Assistant-Host"
)

type MenuItem struct {
	ID        int
	Label     string
	Checked   bool
	Disabled  bool
	Separator bool
}

type HotkeySpec struct {
	Mods  uint32
	VK    uint32
	Label string
}

type HostCallbacks struct {
	Tick            func(now time.Time) bool
	Hotkey          func()
	TrayLeftClick   func()
	TrayMenu        func() []MenuItem
	MenuCommand     func(id int)
	DisplayChange   func()
	SettingsCommand func()
}

type Host struct {
	cb HostCallbacks

	mu            sync.Mutex
	pendingHotkey *HotkeySpec
	candidates    []HotkeySpec
	chosenHotkey  string
	hotkeyOk      bool

	hwnd             uintptr
	classAtom        uint16
	trayIcon         uintptr
	trayAdded        bool
	trayRequested    bool
	trayTip          string
	baseTip          string
	initialized      bool
	stage            string
	toastHold        time.Duration
	toast            *toast
	lastLeftClick    time.Time
	lastRightClick   time.Time
	animRunning      bool
	hotkeyMods       uint32
	hotkeyVK         uint32
	hotkeyRegistered bool
	taskbarMsg       uint32
	wndProcPtr       uintptr

	ready   chan error
	stopped chan struct{}
	stopMu  sync.Mutex
	closing bool
}

func NewHost() *Host {
	return &Host{
		ready:   make(chan error, 1),
		stopped: make(chan struct{}),
		trayTip: buildinfo.AppName,
		baseTip: buildinfo.AppName,
	}
}

func (h *Host) SetCallbacks(cb HostCallbacks) { h.cb = cb }

func (h *Host) Start() error {
	go func() {
		runtime.LockOSThread()
		if err := h.init(); err != nil {
			h.ready <- err
			return
		}
		h.ready <- nil
		h.loop()
		close(h.stopped)
	}()
	if err := <-h.ready; err != nil {
		return err
	}
	return nil
}

func (h *Host) init() error {
	if os.Getenv("SIDEBAR_DEBUG") == "1" {
		defer func(t time.Time) {
			log.Printf("[perf] host.init total=%v", time.Since(t))
		}(time.Now())
	}
	hinst := win32.GetModuleHandle()
	h.wndProcPtr = syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		return h.wndProc(hwnd, uint32(msg), wParam, lParam)
	})

	className := win32.UTF16(hostClassName)
	wc := win32.WNDCLASSEX{
		LpfnWndProc:   h.wndProcPtr,
		HInstance:     hinst,
		LpszClassName: className,
	}
	atom, err := win32.RegisterClassEx(&wc)
	if err != nil {
		return errors.New("注册宿主窗口类失败: " + err.Error())
	}
	h.classAtom = atom

	h.hwnd = win32.CreateWindowEx(
		win32.WS_EX_TOOLWINDOW, className, win32.UTF16("ZhiAi-Assistant"),
		win32.WS_POPUP, 0, 0, 0, 0, 0, 0, hinst, 0)
	if h.hwnd == 0 {
		if e := syscall.GetLastError(); e != nil {
			return errors.New("创建宿主窗口失败: " + e.Error())
		}
		return errors.New("创建宿主窗口失败")
	}

	enableDarkMenus(h.hwnd)

	h.taskbarMsg = win32.RegisterWindowMessage(win32.UTF16("TaskbarCreated"))
	h.trayIcon = createTrayIcon(trayIconSize())

	h.mu.Lock()
	spec := h.pendingHotkey
	cands := h.candidates
	h.mu.Unlock()
	if len(cands) > 0 {
		h.registerFirstAvailable(cands)
	} else if spec != nil {
		h.applyHotkey(*spec)
	}

	win32.SetTimer(h.hwnd, timerCursor, 100, 0)
	return nil
}

func (h *Host) loop() {
	var msg win32.MSG
	for {
		ret := win32.GetMessage(&msg, 0)
		if ret == 0 || ret == -1 {
			break
		}
		win32.TranslateMessage(&msg)
		win32.DispatchMessage(&msg)
	}
	h.cleanup()
}

func (h *Host) cleanup() {
	if h.hotkeyRegistered {
		win32.UnregisterHotKey(h.hwnd, hotkeyIDMain)
		h.hotkeyRegistered = false
	}
	h.destroyToast()
	if h.trayAdded {
		nid := h.trayData()
		win32.ShellNotifyIcon(win32.NIM_DELETE, &nid)
		h.trayAdded = false
	}
	if h.trayIcon != 0 {
		win32.DestroyIcon(h.trayIcon)
		h.trayIcon = 0
	}
}

func (h *Host) Stop() {
	h.stopMu.Lock()
	if h.closing {
		h.stopMu.Unlock()
		return
	}
	h.closing = true
	h.stopMu.Unlock()
	if h.hwnd != 0 {
		win32.PostMessage(h.hwnd, win32.WM_CLOSE, 0, 0)
	}
	select {
	case <-h.stopped:
	case <-time.After(1500 * time.Millisecond):
	}
}

func (h *Host) wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win32.WM_TRAYICON:
		switch uint32(lParam) & 0xffff {
		case win32.WM_LBUTTONUP, win32.NIN_SELECT, win32.NIN_KEYSELECT:
			if !h.Ready() {
				win32.PostMessage(hwnd, startupMsg, 0, 0)
				return 0
			}
			if h.throttled(&h.lastLeftClick) {
				win32.PostMessage(hwnd, trayClickMsg, 0, 0)
			}
		case win32.WM_RBUTTONUP, win32.WM_CONTEXTMENU:
			if !h.Ready() {
				win32.PostMessage(hwnd, startupMsg, 0, 0)
				return 0
			}
			if h.throttled(&h.lastRightClick) {
				win32.PostMessage(hwnd, trayOpenMsg, 0, 0)
			}
		}
		return 0
	case trayClickMsg:
		if h.cb.TrayLeftClick != nil {
			h.cb.TrayLeftClick()
		}
		return 0
	case trayOpenMsg:
		h.showTrayMenu()
		return 0
	case win32.WM_HOTKEY:
		if !h.Ready() {
			return 0
		}
		if h.cb.Hotkey != nil {
			h.cb.Hotkey()
		}
		return 0
	case win32.WM_TIMER:
		if wParam == timerCursor || wParam == timerAnim {
			h.runTick()
		}
		if wParam == toastTimerID {
			h.tickToast()
		}
		return 0
	case startupMsg:
		h.applyStartup()
		return 0
	case kickMessage:
		h.runTick()
		return 0
	case hotkeyMsg:
		h.mu.Lock()
		spec := h.pendingHotkey
		h.mu.Unlock()
		if spec != nil {
			h.applyHotkey(*spec)
		}
		return 0
	case settingsMsg:
		if h.cb.SettingsCommand != nil {
			h.cb.SettingsCommand()
		}
		return 0
	case trayMsg:
		h.mu.Lock()
		requested := h.trayRequested
		h.mu.Unlock()
		if requested && !h.trayAdded {
			_ = h.addTray()
		}
		return 0
	case win32.WM_DISPLAYCHANGE, win32.WM_SETTINGCHANGE:
		if h.cb.DisplayChange != nil {
			h.cb.DisplayChange()
		}
		return 0
	case win32.WM_CLOSE:
		win32.DestroyWindow(h.hwnd)
		return 0
	case win32.WM_DESTROY:
		win32.KillTimer(h.hwnd, timerCursor)
		win32.KillTimer(h.hwnd, timerAnim)
		win32.PostQuitMessage(0)
		return 0
	}
	if h.taskbarMsg != 0 && msg == h.taskbarMsg {
		h.mu.Lock()
		requested := h.trayRequested
		h.mu.Unlock()
		if requested {
			h.trayAdded = false
			_ = h.addTray()
		}
		return 0
	}
	return win32.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (h *Host) throttled(last *time.Time) bool {
	now := time.Now()
	if now.Sub(*last) < trayDedupe {
		return false
	}
	*last = now
	return true
}

func (h *Host) runTick() {
	now := time.Now()
	anim := false
	if h.cb.Tick != nil {
		anim = h.cb.Tick(now)
	}
	if anim && !h.animRunning {
		win32.SetTimer(h.hwnd, timerAnim, 16, 0)
		h.animRunning = true
	} else if !anim && h.animRunning {
		win32.KillTimer(h.hwnd, timerAnim)
		h.animRunning = false
	}
}

func (h *Host) Kick() {
	if h.hwnd != 0 {
		win32.PostMessage(h.hwnd, kickMessage, 0, 0)
	}
}

func (h *Host) trayData() win32.NOTIFYICONDATA {
	var nid win32.NOTIFYICONDATA
	nid.HWnd = h.hwnd
	nid.UID = 1
	nid.UFlags = win32.NIF_MESSAGE | win32.NIF_ICON | win32.NIF_TIP
	nid.UCallbackMessage = win32.WM_TRAYICON
	nid.HIcon = h.trayIcon
	copyTip(&nid.SzTip, h.trayTip)
	return nid
}

func copyTip(dst *[128]uint16, s string) {
	runes := win32.UTF16FromString(s)
	for i := 0; i < len(dst) && i < len(runes); i++ {
		dst[i] = runes[i]
	}
}

func (h *Host) addTray() error {
	if os.Getenv("SIDEBAR_DEBUG") == "1" {
		log.Printf("[perf] addTray called")
	}
	nid := h.trayData()
	if err := win32.ShellNotifyIcon(win32.NIM_ADD, &nid); err != nil {
		return errors.New("添加托盘图标失败")
	}
	h.trayAdded = true
	ver := h.trayData()
	ver.UFlags = 0
	ver.UVersion = win32.NOTIFYICON_VERSION_4
	_ = win32.ShellNotifyIcon(win32.NIM_SETVERSION, &ver)
	return nil
}

func (h *Host) AddTray() {
	h.mu.Lock()
	h.trayRequested = true
	h.mu.Unlock()
	if h.hwnd != 0 {
		win32.PostMessage(h.hwnd, trayMsg, 0, 0)
	}
}

func (h *Host) SetTip(tip string) {
	h.mu.Lock()
	h.baseTip = tip
	if !h.initialized {
		h.mu.Unlock()
		return
	}
	h.trayTip = tip
	added := h.trayAdded
	h.mu.Unlock()
	if !added {
		return
	}
	nid := h.trayData()
	win32.ShellNotifyIcon(win32.NIM_MODIFY, &nid)
}

func cursorPoint() (int32, int32) {
	var p win32.POINT
	win32.GetCursorPos(&p)
	return p.X, p.Y
}

func (h *Host) SetHotkey(spec HotkeySpec) {
	h.mu.Lock()
	h.pendingHotkey = &spec
	h.candidates = nil
	h.mu.Unlock()
	if h.hwnd != 0 {
		win32.PostMessage(h.hwnd, hotkeyMsg, 0, 0)
	}
}

func (h *Host) SetHotkeyCandidates(specs []HotkeySpec) {
	h.mu.Lock()
	h.candidates = specs
	h.mu.Unlock()
}

func (h *Host) registerFirstAvailable(specs []HotkeySpec) {
	for _, spec := range specs {
		if err := win32.RegisterHotKey(h.hwnd, hotkeyIDMain, spec.Mods|win32.MOD_NOREPEAT, spec.VK); err == nil {
			h.hotkeyRegistered = true
			h.mu.Lock()
			h.hotkeyOk = true
			h.chosenHotkey = spec.Label
			h.mu.Unlock()
			return
		}
	}
	h.mu.Lock()
	h.hotkeyOk = false
	h.chosenHotkey = ""
	h.mu.Unlock()
	h.SetTip(i18n.TF("%s（热键被占用）", i18n.T(buildinfo.AppName)))
}

func (h *Host) applyHotkey(spec HotkeySpec) {
	if h.hotkeyRegistered {
		win32.UnregisterHotKey(h.hwnd, hotkeyIDMain)
		h.hotkeyRegistered = false
	}
	if err := win32.RegisterHotKey(h.hwnd, hotkeyIDMain, spec.Mods|win32.MOD_NOREPEAT, spec.VK); err != nil {
		h.mu.Lock()
		h.hotkeyOk = false
		h.mu.Unlock()
		h.SetTip(i18n.TF("%s（热键注册失败）", i18n.T(buildinfo.AppName)))
		return
	}
	h.hotkeyRegistered = true
	h.mu.Lock()
	h.hotkeyOk = true
	h.chosenHotkey = spec.Label
	h.mu.Unlock()
	h.hotkeyMods = spec.Mods
	h.hotkeyVK = spec.VK
}

func (h *Host) HotkeyOK() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hotkeyOk
}

func (h *Host) HotkeyLabel() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.chosenHotkey
}

func trayIconSize() int32 {
	size := win32.GetSystemMetrics(win32.SM_CXSMICON)
	if size <= 0 {
		size = 16
	}
	return size
}

func createTrayIcon(size int32) uintptr {
	src, err := png.Decode(bytes.NewReader(trayIconPNG))
	if err != nil {
		return 0
	}
	out := scaleSquare(src, int(size))
	hdc := win32.GetDC(0)
	if hdc == 0 {
		return 0
	}
	defer win32.ReleaseDC(0, hdc)

	bi := win32.BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(win32.BITMAPINFOHEADER{})),
		BiWidth:       size,
		BiHeight:      -size,
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: win32.BI_RGB,
	}
	var bits unsafe.Pointer
	bmp := win32.CreateDIBSection(hdc, &bi, &bits)
	if bmp == 0 || bits == nil {
		return 0
	}
	px := unsafe.Slice((*byte)(bits), int(size*size*4))
	for y := 0; y < int(size); y++ {
		for x := 0; x < int(size); x++ {
			c := out.NRGBAAt(x, y)
			i := (y*int(size) + x) * 4
			px[i] = c.B
			px[i+1] = c.G
			px[i+2] = c.R
			px[i+3] = c.A
		}
	}

	maskBits := make([]byte, ((int(size)+31)/32)*4*int(size))
	var maskPtr unsafe.Pointer
	if len(maskBits) > 0 {
		maskPtr = unsafe.Pointer(&maskBits[0])
	}
	mask := win32.CreateBitmap(size, size, 1, 1, maskPtr)
	ii := win32.ICONINFO{FIcon: 1, HbmMask: mask, HbmColor: bmp}
	icon := win32.CreateIconIndirect(&ii)
	if mask != 0 {
		win32.DeleteObject(mask)
	}
	win32.DeleteObject(bmp)
	return icon
}

func scaleSquare(src image.Image, size int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= size && sh <= size {
		ox, oy := (size-sw)/2, (size-sh)/2
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				out.SetNRGBA(ox+x, oy+y, color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA))
			}
		}
		return out
	}
	scaleX, scaleY := float64(sw)/float64(size), float64(sh)/float64(size)
	for y := 0; y < size; y++ {
		sy0, sy1 := float64(y)*scaleY, float64(y+1)*scaleY
		for x := 0; x < size; x++ {
			sx0, sx1 := float64(x)*scaleX, float64(x+1)*scaleX
			var rAcc, gAcc, bAcc, aAcc, wSum float64
			for sy := int(sy0); sy < b.Max.Y && float64(sy) < sy1; sy++ {
				wy := math.Min(float64(sy+1), sy1) - math.Max(float64(sy), sy0)
				for sx := int(sx0); sx < b.Max.X && float64(sx) < sx1; sx++ {
					w := (math.Min(float64(sx+1), sx1) - math.Max(float64(sx), sx0)) * wy
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					a := float64(ca) / 65535
					rAcc += float64(cr>>8) * a * w
					gAcc += float64(cg>>8) * a * w
					bAcc += float64(cb>>8) * a * w
					aAcc += float64(ca>>8) * w
					wSum += w
				}
			}
			if wSum == 0 {
				continue
			}
			px := out.Pix[y*out.Stride+x*4:]
			if aAcc > 0 {
				px[0] = clampByte(rAcc * 255 / aAcc)
				px[1] = clampByte(gAcc * 255 / aAcc)
				px[2] = clampByte(bAcc * 255 / aAcc)
			}
			px[3] = clampByte(aAcc / wSum)
		}
	}
	return out
}

func clampByte(v float64) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}

func NotifySettingsCommand() bool {
	hwnd := win32.FindWindow(win32.UTF16(hostClassName), nil)
	if hwnd == 0 {
		return false
	}
	win32.PostMessage(hwnd, settingsMsg, 0, 0)
	return true
}

func SidebarRunning() bool {
	return win32.FindWindow(win32.UTF16(hostClassName), nil) != 0
}

func FocusExistingSettingsWindow() bool {
	hwnd := win32.FindWindow(win32.UTF16("ZhiAi-Assistant-Settings"), nil)
	if hwnd == 0 {
		return false
	}
	win32.ShowWindow(hwnd, win32.SW_RESTORE)
	win32.BringWindowToTop(hwnd)
	win32.SetForegroundWindow(hwnd)
	return true
}
