package vnqr

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Info là nội dung giải mã từ chuỗi VietQR. Trường văn bản rỗng / Amount = 0 nghĩa
// là chuỗi không có tag tương ứng.
type Info struct {
	BankBIN        string
	Account        string
	Service        Service
	Dynamic        bool
	Amount         int64
	Content        string // 62-08
	BillNumber     string // 62-01
	ReferenceLabel string // 62-05
	MerchantName   string // 59
	MerchantCity   string // 60
	Currency       string // 53
	Country        string // 58
	CRC            string // 63
	CRCValid       bool
	Tags           map[string]string // toàn bộ tag cấp ngoài
}

// Bank tra ngân hàng theo BIN trong danh mục.
func (i *Info) Bank() (Bank, bool) { return FindBank(i.BankBIN) }

// Parse giải mã chuỗi VietQR, trả lỗi bọc ErrInvalidCRC / ErrInvalidTLV /
// ErrNotVietQR (kiểm tra bằng errors.Is).
func Parse(qr string) (*Info, error) { return parse(qr, true) }

// ParseLenient giống Parse nhưng không bắt buộc CRC đúng; xem Info.CRCValid.
func ParseLenient(qr string) (*Info, error) { return parse(qr, false) }

func parse(qr string, strictCRC bool) (*Info, error) {
	qr = strings.TrimSpace(qr)
	crcValid := VerifyCRC(qr)
	if strictCRC && !crcValid {
		return nil, ErrInvalidCRC
	}
	tags, err := ParseTLV(qr)
	if err != nil {
		return nil, err
	}
	raw38, ok := tags["38"]
	if !ok {
		return nil, fmt.Errorf("%w: thiếu tag 38", ErrNotVietQR)
	}
	merchant, err := ParseTLV(raw38)
	if err != nil {
		return nil, err
	}
	orgRaw, hasOrg := merchant["01"]
	if strings.ToUpper(merchant["00"]) != NapasGUID || !hasOrg {
		return nil, fmt.Errorf("%w: sai GUID NAPAS", ErrNotVietQR)
	}
	org, err := ParseTLV(orgRaw)
	if err != nil {
		return nil, err
	}
	if org["00"] == "" {
		return nil, fmt.Errorf("%w: thiếu BIN ngân hàng", ErrNotVietQR)
	}
	additional := map[string]string{}
	if raw62, ok := tags["62"]; ok {
		if additional, err = ParseTLV(raw62); err != nil {
			return nil, err
		}
	}
	var amount int64
	if s := tags["54"]; s != "" {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || f < 0 || math.IsInf(f, 0) {
			return nil, fmt.Errorf("%w: số tiền %q không hợp lệ", ErrInvalidTLV, s)
		}
		amount = int64(f)
	}
	service := ServiceAccount
	if strings.ToUpper(merchant["02"]) == ServiceCodeCard {
		service = ServiceCard
	}
	return &Info{
		BankBIN:        org["00"],
		Account:        org["01"],
		Service:        service,
		Dynamic:        tags["01"] == "12",
		Amount:         amount,
		Content:        additional["08"],
		BillNumber:     additional["01"],
		ReferenceLabel: additional["05"],
		MerchantName:   tags["59"],
		MerchantCity:   tags["60"],
		Currency:       tags["53"],
		Country:        tags["58"],
		CRC:            tags["63"],
		CRCValid:       crcValid,
		Tags:           tags,
	}, nil
}
