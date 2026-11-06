package platform

import (
	"math"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform/win32"
)

const (
	toastClass   = "ZhiAi-Assistant-Toast"
	toastTimerID = 3

	toastHeight = 38
	toastPadX   = 13
	toastPadR   = 14
	toastGap    = 9
	toastFontPx = 12
	toastCorner = 10
	toastMinW   = 150
	toastMargin = 14

	toastFadeInStep  = 46
	toastFadeOutStep = 26
	toastTickMs      = 16
	toastDotTicks    = 14

	toastReadyHold = 900 * time.Millisecond
	toastFailHold  = 1800 * time.Millisecond
)

const (
	toastPhaseHidden = iota
	toastPhaseIn
	toastPhaseHold
	toastPhaseOut
)

var (
	toastClassAtom uint16
	toastProcPtr   uintptr
)

type toast struct {
	hwnd uintptr
	font uintptr

	text string
	live bool

	w, h  int32
	alpha int32
	phase int
	ticks int
	dots  int

	hold      time.Duration
	holdUntil time.Time
}

func ensureToastClass() bool {
	if toastClassAtom != 0 {
		return true
	}
	toastProcPtr = syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		return win32.DefWindowProc(hwnd, uint32(msg), wParam, lParam)
	})
	wc := win32.WNDCLASSEX{
		LpfnWndProc:   toastProcPtr,
		HInstance:     win32.GetModuleHandle(),
		LpszClassName: win32.UTF16(toastClass),
	}
	atom, err := win32.RegisterClassEx(&wc)
	if err != nil {
		return false
	}
	toastClassAtom = atom
	return true
}

func (h *Host) toastScale() float64 {
	dpi := win32.GetDpiForWindow(h.hwnd)
	if dpi == 0 {
		dpi = 96
	}
	return float64(dpi) / 96
}

func (h *Host) ensureToast() *toast {
	if h.toast != nil {
		return h.toast
	}
	if !ensureToastClass() {
		return nil
	}
	scale := h.toastScale()
	t := &toast{
		h:    int32(math.Round(toastHeight * scale)),
		w:    int32(math.Round(toastMinW * scale)),
		font: createToastFont(scale),
	}
	x, y := h.toastAnchor(t.w, t.h)
	t.hwnd = win32.CreateWindowEx(
		win32.WS_EX_LAYERED|win32.WS_EX_TOOLWINDOW|win32.WS_EX_TOPMOST|win32.WS_EX_NOACTIVATE|win32.WS_EX_TRANSPARENT,
		win32.UTF16(toastClass), win32.UTF16("Toast"),
		win32.WS_POPUP, x, y, t.w, t.h, h.hwnd, 0, win32.GetModuleHandle(), 0)
	if t.hwnd == 0 {
		return nil
	}
	h.toast = t
	return t
}

func createToastFont(scale float64) uintptr {
	lf := win32.LOGFONTW{
		Height:  -int32(math.Round(float64(toastFontPx) * scale)),
		Weight:  win32.FW_NORMAL,
		CharSet: win32.DEFAULT_CHARSET,
		Quality: win32.CLEARTYPE_QUALITY,
	}
	copy(lf.FaceName[:], win32.UTF16FromString("Microsoft YaHei UI"))
	return win32.CreateFontIndirectW(&lf)
}

func (h *Host) toastAnchor(w, hh int32) (int32, int32) {
	scale := h.toastScale()
	margin := int32(math.Round(toastMargin * scale))
	cx, cy := cursorPoint()
	var info win32.MONITORINFO
	info.CbSize = uint32(unsafe.Sizeof(info))
	mon := win32.MonitorFromPoint(win32.POINT{X: cx, Y: cy}, win32.MONITOR_DEFAULTTONEAREST)
	if win32.GetMonitorInfo(mon, &info) {
		return info.RcWork.Right - w - margin, info.RcWork.Bottom - hh - margin
	}
	sw := win32.GetSystemMetrics(win32.SM_CXSCREEN)
	sh := win32.GetSystemMetrics(win32.SM_CYSCREEN)
	return sw - w - margin, sh - hh - margin
}

func (h *Host) toastTextWidth(t *toast, text string) int32 {
	dc := win32.GetDC(h.hwnd)
	if dc == 0 {
		return int32(len([]rune(text))) * 7
	}
	defer win32.ReleaseDC(h.hwnd, dc)
	old := win32.SelectObject(dc, t.font)
	u := win32.UTF16FromString(text)
	var size win32.SIZE
	win32.TextExtent(dc, &u[0], int32(len(u)-1), &size)
	win32.SelectObject(dc, old)
	return size.Cx
}

func (h *Host) toastResize(t *toast, text string) {
	scale := h.toastScale()
	width := h.toastTextWidth(t, text+"...") + int32(math.Round((toastPadX+toastGap+toastPadR)*scale))
	if min := int32(math.Round(toastMinW * scale)); width < min {
		width = min
	}
	if width == t.w {
		return
	}
	t.w = width
	x, y := h.toastAnchor(t.w, t.h)
	win32.SetWindowPos(t.hwnd, 0, x, y, t.w, t.h, win32.SWP_NOZORDER|win32.SWP_NOACTIVATE)
}

func (h *Host) toastReveal(t *toast) {
	if t.phase == toastPhaseHidden {
		t.alpha = 0
		t.ticks = 0
		win32.ShowWindow(t.hwnd, win32.SW_SHOWNOACTIVATE)
	}
	if t.phase == toastPhaseHidden || t.phase == toastPhaseOut {
		t.phase = toastPhaseIn
	}
}

func (h *Host) toastShow(text string) {
	t := h.ensureToast()
	if t == nil {
		return
	}
	t.text = text
	t.live = true
	t.hold = 0
	t.holdUntil = time.Time{}
	h.toastResize(t, text)
	h.toastReveal(t)
	h.toastRender(t)
	h.startToastTimer()
}

func (h *Host) toastSettle(text string, hold time.Duration) {
	t := h.ensureToast()
	if t == nil {
		return
	}
	t.text = text
	t.live = false
	t.hold = hold
	h.toastResize(t, text)
	h.toastReveal(t)
	if t.phase != toastPhaseIn {
		t.phase = toastPhaseHold
		t.holdUntil = time.Now().Add(hold)
	}
	h.toastRender(t)
	h.startToastTimer()
}

func (h *Host) startToastTimer() {
	if h.toast != nil && h.toast.phase != toastPhaseHidden {
		win32.SetTimer(h.hwnd, toastTimerID, toastTickMs, 0)
	}
}

func (h *Host) tickToast() {
	t := h.toast
	if t == nil {
		win32.KillTimer(h.hwnd, toastTimerID)
		return
	}
	t.ticks++
	switch t.phase {
	case toastPhaseIn:
		t.alpha += toastFadeInStep
		if t.alpha >= 255 {
			t.alpha = 255
			t.phase = toastPhaseHold
			if !t.live && t.hold > 0 {
				t.holdUntil = time.Now().Add(t.hold)
			}
		}
	case toastPhaseHold:
		if t.live {
			if t.ticks%toastDotTicks == 0 {
				t.dots = (t.dots + 1) % 4
			}
		} else if t.holdUntil.IsZero() || time.Now().After(t.holdUntil) {
			t.phase = toastPhaseOut
		}
	case toastPhaseOut:
		t.alpha -= toastFadeOutStep
		if t.alpha <= 0 {
			t.alpha = 0
			t.phase = toastPhaseHidden
			win32.ShowWindow(t.hwnd, win32.SW_HIDE)
			win32.KillTimer(h.hwnd, toastTimerID)
			return
		}
	}
	h.toastRender(t)
}

func (h *Host) toastRender(t *toast) {
	if t.hwnd == 0 || t.w <= 0 || t.h <= 0 {
		return
	}
	scale := h.toastScale()
	w, hh := t.w, t.h
	corner := int32(math.Round(toastCorner * scale))

	screen := win32.GetDC(0)
	mem := win32.CreateCompatibleDC(screen)
	win32.ReleaseDC(0, screen)
	if mem == 0 {
		return
	}
	defer win32.DeleteDC(mem)

	bi := win32.BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(win32.BITMAPINFOHEADER{})),
		BiWidth:       w,
		BiHeight:      -hh,
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0,
	}
	var bits unsafe.Pointer
	dib := win32.CreateDIBSection(mem, &bi, &bits)
	if dib == 0 || bits == nil {
		return
	}
	old := win32.SelectObject(mem, dib)
	defer func() {
		win32.SelectObject(mem, old)
		win32.DeleteObject(dib)
	}()

	px := unsafe.Slice((*uint32)(bits), int(w*hh))
	bg := panelPixel(0x14, 0x18, 0x20, 255)
	edge := panelPixel(0x33, 0x3c, 0x4d, 255)
	for y := int32(0); y < hh; y++ {
		for x := int32(0); x < w; x++ {
			if !insideRounded(x, y, w, hh, corner) {
				px[y*w+x] = 0
				continue
			}
			border := !insideRounded(x-1, y, w, hh, corner) || !insideRounded(x+1, y, w, hh, corner) ||
				!insideRounded(x, y-1, w, hh, corner) || !insideRounded(x, y+1, w, hh, corner)
			if border {
				px[y*w+x] = edge
			} else {
				px[y*w+x] = bg
			}
		}
	}

	fill := func(rc win32.RECT, color uint32) {
		brush := win32.CreateSolidBrush(color)
		if brush == 0 {
			return
		}
		win32.FillRect(mem, &rc, brush)
		win32.DeleteObject(brush)
	}

	trackH := int32(math.Max(2, math.Round(2*scale)))
	trackInset := int32(math.Round(5 * scale))
	trackTop := hh - trackH - trackInset

	barW := int32(math.Max(2, math.Round(3*scale)))
	barX := int32(math.Round(toastPadX * scale))
	barH := int32(math.Round(14 * scale))
	textBottom := hh
	if t.live {
		textBottom = hh - trackInset
	}
	barY := (textBottom - barH) / 2
	fill(win32.RECT{Left: barX, Top: barY, Right: barX + barW, Bottom: barY + barH}, panelAccent)

	trackL := barX + barW + int32(math.Round(toastGap*scale))
	trackR := w - int32(math.Round(toastPadR*scale))
	if t.live {
		fill(win32.RECT{Left: trackL, Top: trackTop, Right: trackR, Bottom: trackTop + trackH},
			panelColor(0x1f, 0x27, 0x34))
		segW := (trackR - trackL) / 4
		if segW < 8 {
			segW = 8
		}
		span := (trackR - trackL) + segW
		off := int32(t.ticks) * 6 % span
		segL := trackL + off - segW
		segR := trackL + off
		if segL < trackL {
			segL = trackL
		}
		if segR > trackR {
			segR = trackR
		}
		if segR > segL {
			fill(win32.RECT{Left: segL, Top: trackTop, Right: segR, Bottom: trackTop + trackH}, panelAccent)
		}
	}

	win32.SelectObject(mem, t.font)
	win32.SetBkMode(mem, win32.TRANSPARENT)
	win32.SetTextColor(mem, panelText)

	label := t.text
	if t.live {
		label += strings.Repeat(".", t.dots)
	}
	textRect := win32.RECT{Left: trackL, Top: 0, Right: trackR, Bottom: textBottom}
	u := win32.UTF16FromString(label)
	win32.DrawText(mem, &u[0], int32(len(u)-1), &textRect,
		win32.DT_LEFT|win32.DT_VCENTER|win32.DT_SINGLELINE|win32.DT_NOPREFIX)

	for y := int32(0); y < hh; y++ {
		for x := int32(0); x < w; x++ {
			i := y*w + x
			if insideRounded(x, y, w, hh, corner) {
				px[i] |= 0xff000000
			} else {
				px[i] = 0
			}
		}
	}

	var rc win32.RECT
	if !win32.GetWindowRect(t.hwnd, &rc) {
		return
	}
	dst := win32.POINT{X: rc.Left, Y: rc.Top}
	size := win32.SIZE{Cx: w, Cy: hh}
	src := win32.POINT{X: 0, Y: 0}
	blend := win32.BLENDFUNCTION{BlendOp: 0, BlendFlags: 0, SourceConstantAlpha: byte(t.alpha), AlphaFormat: 1}
	win32.UpdateLayeredWindow(t.hwnd, 0, &dst, &size, mem, &src, &blend, 0x02)
}

func (h *Host) destroyToast() {
	if h.toast == nil {
		return
	}
	win32.KillTimer(h.hwnd, toastTimerID)
	if h.toast.font != 0 {
		win32.DeleteObject(h.toast.font)
	}
	if h.toast.hwnd != 0 {
		win32.DestroyWindow(h.toast.hwnd)
	}
	h.toast = nil
}

func (h *Host) BeginStartup(text string) {
	h.mu.Lock()
	h.initialized = false
	h.stage = text
	h.toastHold = 0
	h.trayRequested = true
	h.mu.Unlock()
	h.postStartup()
}

func (h *Host) SetStage(text string) {
	h.mu.Lock()
	if h.initialized {
		h.mu.Unlock()
		return
	}
	h.stage = text
	h.mu.Unlock()
	h.postStartup()
}

func (h *Host) MarkReady() {
	h.mu.Lock()
	h.initialized = true
	h.stage = ""
	h.toastHold = toastReadyHold
	h.mu.Unlock()
	h.postStartup()
}

func (h *Host) FailStartup(text string) {
	h.mu.Lock()
	h.initialized = true
	h.stage = text
	h.toastHold = toastFailHold
	h.mu.Unlock()
	h.postStartup()
}

func (h *Host) Ready() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.initialized
}

func (h *Host) postStartup() {
	if h.hwnd != 0 {
		win32.PostMessage(h.hwnd, startupMsg, 0, 0)
	}
}

func (h *Host) applyStartup() {
	h.mu.Lock()
	ready := h.initialized
	stage := h.stage
	base := h.baseTip
	hold := h.toastHold
	h.mu.Unlock()

	tip := stage
	if ready {
		tip = base
	}
	if tip != "" {
		h.mu.Lock()
		h.trayTip = tip
		added := h.trayAdded
		h.mu.Unlock()
		if !added {
			_ = h.addTray()
		} else {
			nid := h.trayData()
			win32.ShellNotifyIcon(win32.NIM_MODIFY, &nid)
		}
	}

	switch {
	case !ready:
		if stage != "" {
			h.toastShow(stage)
		}
	case stage != "":
		h.toastSettle(stage, hold)
	default:
		h.toastSettle(i18n.T("已就绪"), hold)
	}
}
