package platform

func panelColor(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func panelPixel(r, g, b, a byte) uint32 {
	return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

var (
	panelText   = panelColor(0xee, 0xf2, 0xf8)
	panelAccent = panelColor(0x6e, 0xa8, 0xff)
)

func insideRounded(x, y, w, h, r int32) bool {
	if x < 0 || y < 0 || x >= w || y >= h {
		return false
	}
	cx, cy := x, y
	if x < r {
		cx = r
	} else if x >= w-r {
		cx = w - r - 1
	}
	if y < r {
		cy = r
	} else if y >= h-r {
		cy = h - r - 1
	}
	if cx == x || cy == y {
		return true
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}
