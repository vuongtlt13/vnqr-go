package vnqr

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode"
)

//go:embed assets data
var embedded embed.FS

// Bank là một ngân hàng / tổ chức thành viên NAPAS trong danh mục đóng gói sẵn
// (nguồn: https://api.vietqr.io/v2/banks, cập nhật bằng cmd/crawl-banks).
type Bank struct {
	BIN               string // Mã BIN 6 số, ví dụ "970422"
	Code              string // Mã ngắn, ví dụ "MB"
	ShortName         string // Tên ngắn, ví dụ "MBBank"
	Name              string // Tên đầy đủ
	SwiftCode         string
	LogoURL           string
	TransferSupported bool
}

type bankJSON struct {
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	BIN               string  `json:"bin"`
	ShortName         string  `json:"shortName"`
	SwiftCode         *string `json:"swift_code"`
	Logo              *string `json:"logo"`
	TransferSupported *int    `json:"transferSupported"`
}

type bankCatalog struct {
	list  []Bank
	index map[string]Bank
}

var loadBanks = sync.OnceValue(func() bankCatalog {
	data, err := embedded.ReadFile("data/banks.json")
	if err != nil {
		panic("vnqr: thiếu data/banks.json: " + err.Error())
	}
	var raw []bankJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		panic("vnqr: data/banks.json hỏng: " + err.Error())
	}
	cat := bankCatalog{list: make([]Bank, 0, len(raw)), index: make(map[string]Bank)}
	for _, item := range raw {
		b := Bank{
			BIN:               item.BIN,
			Code:              item.Code,
			ShortName:         item.ShortName,
			Name:              item.Name,
			TransferSupported: item.TransferSupported == nil || *item.TransferSupported != 0,
		}
		if b.ShortName == "" {
			b.ShortName = b.Code
		}
		if item.SwiftCode != nil {
			b.SwiftCode = *item.SwiftCode
		}
		if item.Logo != nil {
			b.LogoURL = *item.Logo
		}
		cat.list = append(cat.list, b)
	}
	// Thứ tự ưu tiên: BIN > mã > tên ngắn (không ghi đè khoá đã có).
	for _, field := range []func(Bank) string{
		func(b Bank) string { return b.BIN },
		func(b Bank) string { return b.Code },
		func(b Bank) string { return b.ShortName },
	} {
		for _, b := range cat.list {
			k := bankKey(field(b))
			if _, ok := cat.index[k]; !ok && k != "" {
				cat.index[k] = b
			}
		}
	}
	return cat
})

func bankKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(RemoveDiacritics(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ListBanks trả về bản sao toàn bộ danh mục ngân hàng.
func ListBanks() []Bank {
	return append([]Bank(nil), loadBanks().list...)
}

// FindBank tìm ngân hàng theo BIN ("970422"), mã ("MB", "vcb") hoặc tên ngắn
// ("Vietcombank"). Không phân biệt hoa thường, dấu, khoảng trắng.
func FindBank(query string) (Bank, bool) {
	k := bankKey(query)
	if k == "" {
		return Bank{}, false
	}
	b, ok := loadBanks().index[k]
	return b, ok
}

// ResolveBIN trả về BIN 6 số từ BIN / mã / tên ngân hàng. Chuỗi 6 chữ số được trả
// nguyên (kể cả BIN chưa có trong danh mục).
func ResolveBIN(bank string) (string, error) {
	value := strings.TrimSpace(bank)
	if len(value) == 6 && isDigits([]rune(value)) {
		return value, nil
	}
	if b, ok := FindBank(value); ok {
		return b.BIN, nil
	}
	return "", fmt.Errorf("vnqr: không tìm thấy ngân hàng %q", bank)
}
