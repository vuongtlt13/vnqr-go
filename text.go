package vnqr

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var dReplacer = strings.NewReplacer("đ", "d", "Đ", "D")

// RemoveDiacritics bỏ dấu tiếng Việt và ký tự điều khiển
// ("Thanh toán Đơn" -> "Thanh toan Don").
//
// App ngân hàng hiển thị và đối soát nội dung chuyển khoản dạng không dấu, nên các
// trường văn bản trong VietQR nên được chuẩn hoá bằng hàm này.
func RemoveDiacritics(text string) string {
	if isPrintableASCII(text) {
		return text
	}
	decomposed := norm.NFD.String(dReplacer.Replace(text))
	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.C, r) {
			continue
		}
		b.WriteRune(r)
	}
	return norm.NFC.String(b.String())
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
