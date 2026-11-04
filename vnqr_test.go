package vnqr

import (
	"errors"
	"strings"
	"testing"
)

// Chuỗi tham chiếu sinh từ bản Python vnqr 0.2.0, đảm bảo hai bản ra kết quả giống hệt.
var golden = []struct {
	name string
	p    Payment
	want string
}{
	{
		"legacy",
		Payment{Bank: "970425", Account: "123456789", Amount: 1000000, Content: "V1TH1375842"},
		"00020101021238530010A0000007270123000697042501091234567890208QRIBFTTA5303704540710000005802VN62150811V1TH137584263043158",
	},
	{
		"static",
		Payment{Bank: "VCB", Account: "0011001234567"},
		"00020101021138570010A00000072701270006970436011300110012345670208QRIBFTTA53037045802VN6304E8DB",
	},
	{
		"full",
		Payment{
			Bank: "BIDV", Account: "9704181234567890", Amount: 150000,
			Content: "Thanh toán điện tháng 10", Service: ServiceCard,
			BillNumber: "HD001", ReferenceLabel: "REF-9",
			MerchantName: "Công ty Điện lực", MerchantCity: "Hà Nội",
		},
		"00020101021238600010A00000072701300006970418011697041812345678900208QRIBFTTC530370454061500005802VN5916CONG TY DIEN LUC6006HA NOI62460105HD0010505REF-90824Thanh toan dien thang 1063045DA6",
	},
	{
		"keep diacritics",
		Payment{Bank: "MB", Account: "1", Amount: 1000, Content: "Đơn", KeepDiacritics: true},
		"00020101021238450010A00000072701150006970422010110208QRIBFTTA5303704540410005802VN62070803Đơn6304BBD5",
	},
	{
		"forced static",
		Payment{Bank: "MB", Account: "0123", Amount: 5000, Mode: ModeStatic},
		"00020101021138480010A00000072701180006970422010401230208QRIBFTTA5303704540450005802VN6304D528",
	},
}

func TestBuildMatchesPython(t *testing.T) {
	for _, tc := range golden {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Build(tc.p)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("Build()\n got %s\nwant %s", got, tc.want)
			}
			if !VerifyCRC(got) {
				t.Fatal("CRC không khớp")
			}
		})
	}
}

func TestBuildAcceptsBankCodeAndShortName(t *testing.T) {
	want, _ := Build(Payment{Bank: "970422", Account: "0123", Amount: 5000})
	for _, bank := range []string{"MB", "mbbank", " MB Bank "} {
		got, err := Build(Payment{Bank: bank, Account: "0123", Amount: 5000})
		if err != nil || got != want {
			t.Errorf("Bank %q: got %q, err %v", bank, got, err)
		}
	}
}

func TestBuildRejectsInvalidInput(t *testing.T) {
	for name, p := range map[string]Payment{
		"unknown bank":  {Bank: "khong-co", Account: "1"},
		"empty account": {Bank: "MB", Account: "  "},
		"long account":  {Bank: "MB", Account: strings.Repeat("1", 26)},
		"negative":      {Bank: "MB", Account: "1", Amount: -5},
		"too large":     {Bank: "MB", Account: "1", Amount: 1e13},
		"long content":  {Bank: "MB", Account: "1", Content: strings.Repeat("x", 120)},
		"bad service":   {Bank: "MB", Account: "1", Service: "wallet"},
		"bad mode":      {Bank: "MB", Account: "1", Mode: "maybe"},
	} {
		if _, err := Build(p); err == nil {
			t.Errorf("%s: mong đợi lỗi", name)
		}
	}
}

func TestParseRoundtrip(t *testing.T) {
	info, err := Parse(golden[2].want)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := info.Bank()
	checks := map[string]bool{
		"bin":       info.BankBIN == "970418",
		"bank":      ok && b.Code == "BIDV",
		"account":   info.Account == "9704181234567890",
		"service":   info.Service == ServiceCard,
		"dynamic":   info.Dynamic,
		"amount":    info.Amount == 150000,
		"content":   info.Content == "Thanh toan dien thang 10",
		"bill":      info.BillNumber == "HD001",
		"reference": info.ReferenceLabel == "REF-9",
		"name":      info.MerchantName == "CONG TY DIEN LUC",
		"city":      info.MerchantCity == "HA NOI",
		"currency":  info.Currency == "704" && info.Country == "VN",
		"crc":       info.CRCValid && info.CRC == "5DA6",
	}
	for k, pass := range checks {
		if !pass {
			t.Errorf("sai trường %s: %+v", k, info)
		}
	}

	keep, err := Parse(golden[3].want)
	if err != nil || keep.Content != "Đơn" {
		t.Errorf("giữ dấu: %+v, %v", keep, err)
	}
	static, _ := Parse(golden[1].want)
	if static.Dynamic || static.Amount != 0 {
		t.Errorf("QR tĩnh: %+v", static)
	}
}

func TestParseErrors(t *testing.T) {
	qr := golden[4].want
	broken := qr[:len(qr)-4] + "0000"
	if VerifyCRC(broken) {
		t.Fatal("CRC hỏng vẫn qua")
	}
	if _, err := Parse(broken); !errors.Is(err, ErrInvalidCRC) {
		t.Errorf("mong đợi ErrInvalidCRC, nhận %v", err)
	}
	if info, err := ParseLenient(broken); err != nil || info.CRCValid {
		t.Errorf("ParseLenient: %+v, %v", info, err)
	}
	if _, err := Parse("hello"); err == nil {
		t.Error("chuỗi rác phải lỗi")
	}
	if _, err := ParseTLV("0005abc"); !errors.Is(err, ErrInvalidTLV) {
		t.Errorf("mong đợi ErrInvalidTLV, nhận %v", err)
	}
	notVietQR := "000201" + "5802VN" + "6304"
	notVietQR += CRC16(notVietQR)
	if _, err := Parse(notVietQR); !errors.Is(err, ErrNotVietQR) {
		t.Errorf("mong đợi ErrNotVietQR, nhận %v", err)
	}
}

func TestBanks(t *testing.T) {
	banks := ListBanks()
	if len(banks) < 50 {
		t.Fatalf("danh mục chỉ có %d ngân hàng", len(banks))
	}
	for _, b := range banks {
		if len(b.BIN) != 6 || !isDigits([]rune(b.BIN)) {
			t.Errorf("BIN sai: %+v", b)
		}
	}
	vcb, ok := FindBank("Vietcombank")
	if !ok || vcb.BIN != "970436" {
		t.Fatalf("Vietcombank: %+v", vcb)
	}
	for _, q := range []string{"970436", " vcb ", "VCB"} {
		if b, ok := FindBank(q); !ok || b != vcb {
			t.Errorf("FindBank(%q) = %+v", q, b)
		}
	}
	if _, ok := FindBank(""); ok {
		t.Error("chuỗi rỗng không được khớp")
	}
	if bin, _ := ResolveBIN("tcb"); bin != "970407" {
		t.Errorf("Techcombank: %s", bin)
	}
	if bin, _ := ResolveBIN("970999"); bin != "970999" {
		t.Errorf("BIN ngoài danh mục: %s", bin)
	}
	if _, err := ResolveBIN("khong-co"); err == nil {
		t.Error("mong đợi lỗi")
	}
	banks[0].BIN = "x"
	if ListBanks()[0].BIN == "x" {
		t.Error("ListBanks phải trả bản sao")
	}
}

func TestRemoveDiacritics(t *testing.T) {
	for in, want := range map[string]string{
		"Thanh toán Đơn hàng": "Thanh toan Don hang",
		"Nguyễn Văn Ả":        "Nguyen Van A",
		"abc\n123":            "abc123",
		"ASCII ok":            "ASCII ok",
		"":                    "",
	} {
		if got := RemoveDiacritics(in); got != want {
			t.Errorf("RemoveDiacritics(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCRC16(t *testing.T) {
	if got := CRC16("123456789"); got != "29B1" { // giá trị kiểm chuẩn CRC-16/CCITT-FALSE
		t.Errorf("CRC16 = %s", got)
	}
}
