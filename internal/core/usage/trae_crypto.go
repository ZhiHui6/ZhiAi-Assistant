package usage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"strings"
)

var (
	traeSaltA = []byte{82, 9, 106, 213, 48, 54, 165, 56, 191, 64, 163, 158, 129, 243, 215, 251, 124, 227, 57, 130, 155, 47, 255, 135, 52, 142, 67, 68, 196, 222, 233, 203, 84, 123, 148, 50, 166, 194, 35, 61, 238, 76, 149, 11, 66, 250, 195, 78, 8, 46, 161, 102, 40, 217, 36, 178, 118, 91, 162, 73, 109, 139, 209, 37}
	traeSaltB = []byte{31, 221, 168, 51, 136, 7, 199, 49, 177, 18, 16, 89, 39, 128, 236, 95, 96, 81, 127, 169, 25, 181, 74, 13, 45, 229, 122, 159, 147, 201, 156, 239, 160, 224, 59, 77, 174, 42, 245, 176, 200, 235, 187, 60, 131, 83, 153, 97, 23, 43, 4, 126, 186, 119, 214, 38, 225, 105, 20, 99, 85, 33, 12, 125}
	traeSaltC = []byte{191, 192, 216, 250, 122, 246, 220, 97, 31, 254, 98, 27, 8, 72, 71, 176, 135, 99, 96, 18, 127, 101, 203, 104, 211, 102, 191, 125, 37, 72, 150, 156, 51, 229, 121, 35, 17, 153, 141, 177, 110, 131, 150, 128, 172, 255, 254, 6, 18, 140, 55, 62, 236, 249, 135, 64, 135, 12, 117, 4, 89, 149, 168, 209}
	traeSaltD = []byte{246, 204, 26, 232, 232, 70, 129, 109, 223, 146, 169, 242, 23, 241, 105, 145, 50, 196, 165, 42, 254, 120, 3, 54, 244, 207, 209, 85, 53, 6, 138, 106, 175, 148, 31, 204, 186, 186, 165, 182, 87, 142, 49, 10, 39, 110, 26, 154, 86, 56, 173, 125, 18, 64, 198, 225, 99, 99, 83, 82, 191, 134, 76, 170}
)

func decryptTraeStorageValue(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(raw) <= 102 {
		return "", errors.New("Trae 登录数据长度异常")
	}
	header := raw[:6]
	var salt []byte
	switch {
	case string(header) == string([]byte{0x74, 0x63, 0x05, 0x10, 0x00, 0x00}):
		salt = xorBytes(traeSaltA, traeSaltB)
	case string(header) == string([]byte{18, 57, 32, 32, 2, 3}):
		salt = xorBytes(traeSaltC, traeSaltD)
	default:
		return "", errors.New("Trae 登录数据格式不受支持")
	}
	random := raw[6:38]
	encrypted := raw[38:]
	first := sha512.Sum512(random)
	seed := append(first[:], salt...)
	derived := sha512.Sum512(seed)
	block, err := aes.NewCipher(derived[:16])
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, derived[16:32]).CryptBlocks(plain, encrypted)
	plain, err = pkcs7Unpad(plain)
	if err != nil || len(plain) < 64 {
		return "", errors.New("Trae 登录数据校验失败")
	}
	expected := plain[:64]
	body := plain[64:]
	actual := sha512.Sum512(body)
	if string(expected) != string(actual[:]) {
		return "", errors.New("Trae 登录数据完整性校验失败")
	}
	return string(body), nil
}

func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		var other byte
		if i < len(b) {
			other = b[i]
		}
		out[i] = a[i] ^ other
	}
	return out
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("空数据")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > 16 || padding > len(data) {
		return nil, errors.New("填充错误")
	}
	for _, value := range data[len(data)-padding:] {
		if int(value) != padding {
			return nil, errors.New("填充错误")
		}
	}
	return data[:len(data)-padding], nil
}
