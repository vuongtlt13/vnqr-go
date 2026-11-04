package vnqr

import (
	"errors"
	"strconv"
	"strings"
)

// Hằng số định danh dịch vụ NAPAS247 trong tag 38.
const (
	NapasGUID          = "A000000727"
	ServiceCodeAccount = "QRIBFTTA" // chuyển nhanh đến tài khoản
	ServiceCodeCard    = "QRIBFTTC" // chuyển nhanh đến thẻ
)

// Service là loại đích chuyển tiền.
type Service string

const (
	ServiceAccount Service = "account" // QRIBFTTA (mặc định)
	ServiceCard    Service = "card"    // QRIBFTTC
)

// Mode chọn QR tĩnh hay động (tag 01).
type Mode string

const (
	// ModeAuto (mặc định): QR động nếu có số tiền, số hoá đơn hoặc mã tham chiếu.
	ModeAuto Mode = ""
	// ModeStatic: "11", dùng được nhiều lần.
	ModeStatic Mode = "static"
	// ModeDynamic: "12", dùng cho một giao dịch.
	ModeDynamic Mode = "dynamic"
)

// Payment là thông tin để tạo chuỗi VietQR. Chỉ Bank và Account là bắt buộc.
type Payment struct {
	// Bank: BIN 6 số ("970422"), mã ("MB") hoặc tên ngắn ("MBBank").
	Bank string
	// Account: số tài khoản hoặc số thẻ, tối đa 25 ký tự.
	Account string
	// Amount: số tiền VND. 0 = để người chuyển tự nhập.
	Amount int64
	// Content: nội dung chuyển khoản (tag 62-08).
	Content string
	// Service: ServiceAccount (mặc định) hoặc ServiceCard.
	Service Service
	// BillNumber: số hoá đơn (tag 62-01, tối đa 25 ký tự).
	BillNumber string
	// ReferenceLabel: mã tham chiếu / mã đơn (tag 62-05, tối đa 25 ký tự).
	ReferenceLabel string
	// MerchantName: tên đơn vị thụ hưởng (tag 59, tối đa 25 ký tự, viết hoa).
	MerchantName string
	// MerchantCity: thành phố (tag 60, tối đa 15 ký tự, viết hoa).
	MerchantCity string
	// Mode: ModeAuto (mặc định), ModeStatic hoặc ModeDynamic.
	Mode Mode
	// KeepDiacritics giữ nguyên dấu tiếng Việt ở các trường văn bản. Mặc định các
	// trường được bỏ dấu (khuyến nghị).
	KeepDiacritics bool
}

// Build tạo chuỗi VietQR (EMVCo) kèm CRC.
func Build(p Payment) (string, error) {
	bin, err := ResolveBIN(p.Bank)
	if err != nil {
		return "", err
	}
	account := strings.TrimSpace(p.Account)
	if n := len([]rune(account)); n == 0 || n > 25 {
		return "", errors.New("vnqr: số tài khoản/thẻ không được rỗng và tối đa 25 ký tự")
	}
	if p.Amount < 0 || p.Amount >= 1e13 {
		return "", errors.New("vnqr: số tiền phải là số nguyên dương, tối đa 13 chữ số")
	}
	serviceCode := ServiceCodeAccount
	switch p.Service {
	case ServiceAccount, "":
	case ServiceCard:
		serviceCode = ServiceCodeCard
	default:
		return "", errors.New("vnqr: service phải là \"account\" hoặc \"card\"")
	}

	text := func(value string, limit int) string {
		value = strings.TrimSpace(value)
		if value == "" {
			return ""
		}
		if !p.KeepDiacritics {
			value = RemoveDiacritics(value)
		}
		if r := []rune(value); limit > 0 && len(r) > limit {
			value = string(r[:limit])
		}
		return value
	}

	var org, merchant, additional, payload tlvWriter
	org.add("00", bin)
	org.add("01", account)
	merchant.add("00", NapasGUID)
	merchant.add("01", org.String())
	merchant.add("02", serviceCode)
	for _, f := range []struct{ tag, value string }{
		{"01", text(p.BillNumber, 25)},
		{"05", text(p.ReferenceLabel, 25)},
		{"08", text(p.Content, 0)},
	} {
		if f.value != "" {
			additional.add(f.tag, f.value)
		}
	}

	var dynamic bool
	switch p.Mode {
	case ModeAuto:
		dynamic = p.Amount > 0 || p.BillNumber != "" || p.ReferenceLabel != ""
	case ModeStatic:
	case ModeDynamic:
		dynamic = true
	default:
		return "", errors.New("vnqr: mode phải là \"\", \"static\" hoặc \"dynamic\"")
	}

	payload.add("00", "01")
	if dynamic {
		payload.add("01", "12")
	} else {
		payload.add("01", "11")
	}
	payload.add("38", merchant.String())
	payload.add("53", "704")
	if p.Amount > 0 {
		payload.add("54", strconv.FormatInt(p.Amount, 10))
	}
	payload.add("58", "VN")
	if name := strings.ToUpper(text(p.MerchantName, 25)); name != "" {
		payload.add("59", name)
	}
	if city := strings.ToUpper(text(p.MerchantCity, 15)); city != "" {
		payload.add("60", city)
	}
	if a := additional.String(); a != "" {
		payload.add("62", a)
	}
	for _, w := range []*tlvWriter{&org, &merchant, &additional, &payload} {
		if w.err != nil {
			return "", w.err
		}
	}
	data := payload.String() + "6304"
	return data + CRC16(data), nil
}
