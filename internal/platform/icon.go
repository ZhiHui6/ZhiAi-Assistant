package platform

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/platform/win32"
)

const iconCacheLimit = 600

var (
	iconMu   sync.Mutex
	iconMem  = map[string]string{}
	iconKeys []string
	iconStat = map[string]time.Time{}
)

func IconDataURL(path string) string {
	key := cacheKey(path)
	iconMu.Lock()
	if v, ok := iconMem[key]; ok {
		iconMu.Unlock()
		return v
	}
	iconMu.Unlock()

	if v := readIconCache(key); v != "" {
		iconMu.Lock()
		iconMem[key] = v
		iconKeys = append(iconKeys, key)
		iconMu.Unlock()
		return v
	}

	pngBytes, err := ExtractIconPNG(path)
	if err != nil || len(pngBytes) == 0 {
		return ""
	}
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	writeIconCache(key, pngBytes)

	iconMu.Lock()
	iconMem[key] = url
	iconKeys = append(iconKeys, key)
	if len(iconKeys) > iconCacheLimit {
		drop := iconKeys[0]
		iconKeys = iconKeys[1:]
		delete(iconMem, drop)
	}
	iconMu.Unlock()
	return url
}

func cacheKey(path string) string {
	mtime := ""
	if st, err := os.Stat(path); err == nil {
		mtime = st.ModTime().Format(time.RFC3339Nano)
	}
	sum := sha1.Sum([]byte(path + "|" + mtime))
	return hex.EncodeToString(sum[:])
}

func iconCachePath(key string) string {
	return filepath.Join(store.DataDir(), "cache", "icons", key+".png")
}

func readIconCache(key string) string {
	b, err := os.ReadFile(iconCachePath(key))
	if err != nil || len(b) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
}

func writeIconCache(key string, b []byte) {
	p := iconCachePath(key)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, b, 0o644)
}

func ExtractIconPNG(path string) ([]byte, error) {
	flags := uint32(win32.SHGFI_ICON | win32.SHGFI_LARGEICON)
	var attrs uint32
	if _, err := os.Stat(path); err != nil {
		flags |= win32.SHGFI_USEFILEATTRIBUTES
		attrs = win32.FILE_ATTRIBUTE_NORMAL
		if ext := filepath.Ext(path); ext != "" {
			_ = ext
		}
	}
	info, ok := win32.SHGetFileInfo(path, attrs, flags)
	if !ok || info.HIcon == 0 {
		return nil, errors.New("SHGetFileInfo 未返回图标")
	}
	defer win32.DestroyIcon(info.HIcon)
	return hiconToPNG(info.HIcon)
}

func hiconToPNG(hicon uintptr) ([]byte, error) {
	ii, ok := win32.GetIconInfo(hicon)
	if !ok {
		return nil, errors.New("GetIconInfo 失败")
	}
	if ii.HbmColor != 0 {
		defer win32.DeleteObject(ii.HbmColor)
	}
	if ii.HbmMask != 0 {
		defer win32.DeleteObject(ii.HbmMask)
	}
	if ii.HbmColor == 0 {
		return nil, errors.New("图标无彩色位图")
	}

	var bm win32.BITMAP
	if win32.GetObject(ii.HbmColor, unsafe.Pointer(&bm), int32(unsafe.Sizeof(bm))) == 0 {
		return nil, errors.New("GetObject 失败")
	}
	w, h := int(bm.BmWidth), int(bm.BmHeight)
	if w <= 0 || h <= 0 || w > 512 || h > 512 {
		return nil, errors.New("图标尺寸异常")
	}

	hdc := win32.GetDC(0)
	if hdc == 0 {
		return nil, errors.New("GetDC 失败")
	}
	defer win32.ReleaseDC(0, hdc)

	bi := win32.BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(win32.BITMAPINFOHEADER{})),
		BiWidth:       int32(w),
		BiHeight:      -int32(h),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: win32.BI_RGB,
	}
	buf := make([]byte, w*h*4)
	if win32.GetDIBits(hdc, ii.HbmColor, 0, uint32(h), unsafe.Pointer(&buf[0]), &bi) == 0 {
		return nil, errors.New("GetDIBits 失败")
	}

	hasAlpha := false
	for i := 0; i < w*h; i++ {
		if buf[i*4+3] != 0 {
			hasAlpha = true
			break
		}
	}
	var maskAlpha []byte
	if !hasAlpha && ii.HbmMask != 0 {
		maskBuf := make([]byte, w*h*4)
		mbi := bi
		if win32.GetDIBits(hdc, ii.HbmMask, 0, uint32(h), unsafe.Pointer(&maskBuf[0]), &mbi) > 0 {
			maskAlpha = make([]byte, w*h)
			for i := 0; i < w*h; i++ {
				if maskBuf[i*4] == 0 {
					maskAlpha[i] = 255
				}
			}
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		b := buf[i*4]
		g := buf[i*4+1]
		r := buf[i*4+2]
		a := buf[i*4+3]
		if maskAlpha != nil {
			a = maskAlpha[i]
		}
		if a == 0 {
			img.SetRGBA(i%w, i/w, color.RGBA{})
			continue
		}
		if a != 255 {
			r = uint8(minInt(255, int(r)*255/int(a)))
			g = uint8(minInt(255, int(g)*255/int(a)))
			b = uint8(minInt(255, int(b)*255/int(a)))
		}
		img.SetRGBA(i%w, i/w, color.RGBA{R: r, G: g, B: b, A: a})
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ClearIconCache() {
	iconMu.Lock()
	iconMem = map[string]string{}
	iconKeys = nil
	iconMu.Unlock()
	dir := filepath.Join(store.DataDir(), "cache", "icons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

var _ = iconStat
