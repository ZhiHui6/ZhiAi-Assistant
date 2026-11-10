package platform

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zhi-ai-assistant/internal/core/store"
)

const (
	thumbMaxDim   = 96
	thumbMaxBytes = 24 << 20
	thumbMemLimit = 400
)

var (
	thumbMu   sync.Mutex
	thumbMem  = map[string]string{}
	thumbKeys []string
)

// ThumbDataURL 生成图片缩略图（PNG data URL），非图片或解码失败时返回空串。
func ThumbDataURL(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" {
		return ""
	}
	key := cacheKey(path)
	thumbMu.Lock()
	if v, ok := thumbMem[key]; ok {
		thumbMu.Unlock()
		return v
	}
	thumbMu.Unlock()

	if v := readThumbCache(key); v != "" {
		thumbMu.Lock()
		thumbMem[key] = v
		thumbKeys = append(thumbKeys, key)
		thumbMu.Unlock()
		return v
	}

	st, err := os.Stat(path)
	if err != nil || st.Size() == 0 || st.Size() > thumbMaxBytes {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return ""
	}

	dst := scaleImage(src, thumbMaxDim)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return ""
	}
	writeThumbCache(key, buf.Bytes())
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	thumbMu.Lock()
	thumbMem[key] = url
	thumbKeys = append(thumbKeys, key)
	if len(thumbKeys) > thumbMemLimit {
		drop := thumbKeys[0]
		thumbKeys = thumbKeys[1:]
		delete(thumbMem, drop)
	}
	thumbMu.Unlock()
	return url
}

func scaleImage(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	scale := math.Min(float64(maxDim)/float64(w), float64(maxDim)/float64(h))
	if scale > 1 {
		scale = 1
	}
	dw := int(math.Max(1, math.Round(float64(w)*scale)))
	dh := int(math.Max(1, math.Round(float64(h)*scale)))

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	sx := float64(w) / float64(dw)
	sy := float64(h) / float64(dh)
	for y := 0; y < dh; y++ {
		fy := (float64(y)+0.5)*sy - 0.5
		y0 := int(math.Floor(fy))
		wy := fy - float64(y0)
		y1 := clampIndex(y0+1, h)
		y0 = clampIndex(y0, h)
		for x := 0; x < dw; x++ {
			fx := (float64(x)+0.5)*sx - 0.5
			x0 := int(math.Floor(fx))
			wx := fx - float64(x0)
			x1 := clampIndex(x0+1, w)
			x0 = clampIndex(x0, w)

			r00, g00, b00, a00 := src.At(b.Min.X+x0, b.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := src.At(b.Min.X+x1, b.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := src.At(b.Min.X+x0, b.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := src.At(b.Min.X+x1, b.Min.Y+y1).RGBA()

			r := bilerp(r00, r10, r01, r11, wx, wy)
			g := bilerp(g00, g10, g01, g11, wx, wy)
			bl := bilerp(b00, b10, b01, b11, wx, wy)
			a := bilerp(a00, a10, a01, a11, wx, wy)

			i := dst.PixOffset(x, y)
			dst.Pix[i] = uint8(r >> 8)
			dst.Pix[i+1] = uint8(g >> 8)
			dst.Pix[i+2] = uint8(bl >> 8)
			dst.Pix[i+3] = uint8(a >> 8)
		}
	}
	return dst
}

func clampIndex(v, n int) int {
	if v < 0 {
		return 0
	}
	if v >= n {
		return n - 1
	}
	return v
}

func bilerp(v00, v10, v01, v11 uint32, wx, wy float64) uint32 {
	top := float64(v00)*(1-wx) + float64(v10)*wx
	bottom := float64(v01)*(1-wx) + float64(v11)*wx
	return uint32(top*(1-wy) + bottom*wy)
}

func thumbCachePath(key string) string {
	return filepath.Join(store.DataDir(), "cache", "thumbs", key+".png")
}

func readThumbCache(key string) string {
	b, err := os.ReadFile(thumbCachePath(key))
	if err != nil || len(b) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
}

func writeThumbCache(key string, b []byte) {
	p := thumbCachePath(key)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, b, 0o644)
}
