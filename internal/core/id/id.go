package id

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

func New() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}
