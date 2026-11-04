package vnqr

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Lỗi trả về khi giải mã, dùng với errors.Is.
var (
	// ErrInvalidTLV: chuỗi không đúng cấu trúc TLV (tag 2 số + độ dài 2 số + giá trị).
	ErrInvalidTLV = errors.New("vnqr: TLV không hợp lệ")
	// ErrInvalidCRC: mã kiểm tra CRC (tag 63) không khớp.
	ErrInvalidCRC = errors.New("vnqr: sai mã kiểm tra CRC")
	// ErrNotVietQR: chuỗi đúng TLV nhưng không phải VietQR NAPAS247.
	ErrNotVietQR = errors.New("vnqr: không phải VietQR")
)

// CRC16 tính CRC16-CCITT (đa thức 0x1021, khởi tạo 0xFFFF) dạng 4 ký tự hex in hoa,
// dùng cho tag 63 của VietQR.
func CRC16(data string) string {
	crc := uint16(0xFFFF)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// VerifyCRC kiểm tra chuỗi kết thúc bằng "6304XXXX" và mã CRC16 khớp.
func VerifyCRC(qr string) bool {
	qr = strings.TrimSpace(qr)
	runes := []rune(qr)
	n := len(runes)
	if n < 8 || string(runes[n-8:n-4]) != "6304" {
		return false
	}
	return CRC16(string(runes[:n-4])) == strings.ToUpper(string(runes[n-4:]))
}

// ParseTLV tách một chuỗi TLV (tag 2 số + độ dài 2 số + giá trị) thành map tag ->
// giá trị. Độ dài tính theo ký tự (rune), giống bản Python.
func ParseTLV(data string) (map[string]string, error) {
	runes := []rune(data)
	result := make(map[string]string)
	for pos := 0; pos < len(runes); {
		if pos+4 > len(runes) || !isDigits(runes[pos:pos+4]) {
			return nil, fmt.Errorf("%w tại vị trí %d: %q", ErrInvalidTLV, pos, string(runes[pos:]))
		}
		tag := string(runes[pos : pos+2])
		length := int(runes[pos+2]-'0')*10 + int(runes[pos+3]-'0')
		end := pos + 4 + length
		if end > len(runes) {
			return nil, fmt.Errorf("%w: tag %s khai báo dài %d nhưng chỉ còn %d",
				ErrInvalidTLV, tag, length, len(runes)-pos-4)
		}
		result[tag] = string(runes[pos+4 : end])
		pos = end
	}
	return result, nil
}

func isDigits(rs []rune) bool {
	for _, r := range rs {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// tlvWriter ghép các trường TLV, giữ lỗi đầu tiên.
type tlvWriter struct {
	b   strings.Builder
	err error
}

func (w *tlvWriter) add(tag, value string) {
	if w.err != nil {
		return
	}
	n := utf8.RuneCountInString(value)
	if n > 99 {
		w.err = fmt.Errorf("vnqr: trường %s dài %d ký tự, vượt giới hạn 99", tag, n)
		return
	}
	fmt.Fprintf(&w.b, "%s%02d%s", tag, n, value)
}

func (w *tlvWriter) String() string { return w.b.String() }
