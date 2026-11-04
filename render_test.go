package vnqr

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

var testQR = func() string {
	qr, err := Build(Payment{
		Bank: "MB", Account: "0123456789", Amount: 2000000,
		Content: "Thanh toan DH123", MerchantName: "Nguyen Van A",
	})
	if err != nil {
		panic(err)
	}
	return qr
}()

// decode giải mã ảnh (trải nền trắng nếu trong suốt) bằng gozxing.
func decode(t *testing.T, img image.Image) string {
	t.Helper()
	flat := image.NewRGBA(img.Bounds())
	draw.Draw(flat, flat.Rect, image.White, image.Point{}, draw.Src)
	draw.Draw(flat, flat.Rect, img, img.Bounds().Min, draw.Over)
	bmp, err := gozxing.NewBinaryBitmapFromImage(flat)
	if err != nil {
		t.Fatal(err)
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, map[gozxing.DecodeHintType]any{
		gozxing.DecodeHintType_TRY_HARDER: true,
	})
	if err != nil {
		t.Fatalf("không giải mã được ảnh: %v", err)
	}
	return res.GetText()
}

func mustRender(t *testing.T, content string, opts *RenderOptions) *Image {
	t.Helper()
	img, err := Render(content, opts)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func ptr[T any](v T) *T { return &v }

func TestTemplatesAreScannable(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts *RenderOptions
		w, h int
	}{
		{"default", nil, 500, 500},
		{"size 300", &RenderOptions{Size: 300}, 300, 300},
		{"compact", &RenderOptions{Template: TemplateCompact}, 540, 540},
		{"compact x2", &RenderOptions{Template: TemplateCompact, Scale: 2}, 1080, 1080},
		{"print", &RenderOptions{Template: TemplatePrint}, 600, 746},
		{"print no amount", &RenderOptions{Template: TemplatePrint, Amount: ptr[int64](0)}, 600, 718},
		{"print x1.5", &RenderOptions{Template: TemplatePrint, Scale: 1.5}, 900, 1119},
		{"no logo", &RenderOptions{NoCenterLogo: true}, 500, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := mustRender(t, testQR, tc.opts)
			if b := img.RGBA.Bounds(); b.Dx() != tc.w || b.Dy() != tc.h {
				t.Fatalf("kích thước %v, mong đợi %dx%d", b.Size(), tc.w, tc.h)
			}
			if got := decode(t, img.RGBA); got != testQR {
				t.Fatalf("giải mã ra %q", got)
			}
		})
	}
}

func TestStylesAreScannable(t *testing.T) {
	for name, st := range map[string]Style{
		"rounded":    {ModuleShape: ModuleRounded, EyeShape: EyeRounded, Color: MustHex("#003B70")},
		"circle":     {ModuleShape: ModuleCircle, EyeShape: EyeCircle, Gradient: GradientDiagonal},
		"gapped":     {ModuleShape: ModuleGapped, Gradient: GradientRadial, GradientColor: MustHex("#0050A0")},
		"eye color":  {Gradient: GradientHorizontal, EyeColor: MustHex("#C8102E")},
		"vertical H": {Gradient: GradientVertical, ErrorCorrection: ECHigh},
		"level L":    {ErrorCorrection: ECLow},
	} {
		t.Run(name, func(t *testing.T) {
			for _, tpl := range []Template{TemplateQROnly, TemplateCompact} {
				img := mustRender(t, testQR, &RenderOptions{Template: tpl, Style: st})
				if got := decode(t, img.RGBA); got != testQR {
					t.Fatalf("%s: giải mã ra %q", tpl, got)
				}
			}
		})
	}
}

func TestTransparentBackground(t *testing.T) {
	img := mustRender(t, testQR, &RenderOptions{Style: Style{Transparent: true}})
	if a := img.RGBA.RGBAAt(0, 0).A; a != 0 {
		t.Fatalf("góc ảnh phải trong suốt, alpha = %d", a)
	}
	if got := decode(t, img.RGBA); got != testQR {
		t.Fatalf("giải mã ra %q", got)
	}
	b, err := img.Bytes(FormatJPEG)
	if err != nil {
		t.Fatal(err)
	}
	j, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if r, g, bl, _ := j.At(0, 0).RGBA(); r>>8 < 250 || g>>8 < 250 || bl>>8 < 250 {
		t.Error("JPEG phải trải nền trắng")
	}
}

func TestCustomLogoAndBankOverride(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 40, 20))
	draw.Draw(logo, logo.Rect, image.NewUniform(color.RGBA{200, 0, 0, 255}), image.Point{}, draw.Src)
	img := mustRender(t, testQR, &RenderOptions{
		Template: TemplateCompact, CenterLogo: logo, Bank: "VCB", BankLogo: logo,
	})
	if got := decode(t, img.RGBA); got != testQR {
		t.Fatalf("giải mã ra %q", got)
	}
	img = mustRender(t, testQR, &RenderOptions{CenterLogo: logo, CenterLogoRatio: 0.28})
	if got := decode(t, img.RGBA); got != testQR {
		t.Fatalf("giải mã ra %q", got)
	}
}

func TestUnknownBankFallsBackToText(t *testing.T) {
	qr, err := Build(Payment{Bank: "970499", Account: "123", Amount: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if got := decode(t, mustRender(t, qr, &RenderOptions{Template: TemplatePrint}).RGBA); got != qr {
		t.Fatalf("giải mã ra %q", got)
	}
	// Ngân hàng không có logo đóng gói -> vẽ tên ngắn.
	img := mustRender(t, testQR, &RenderOptions{Template: TemplatePrint, Bank: "VCB", AccountName: strings.Repeat("Nguyễn Văn A ", 8)})
	if got := decode(t, img.RGBA); got != testQR {
		t.Fatalf("giải mã ra %q", got)
	}
}

func TestNonVietQRContentRenders(t *testing.T) {
	img := mustRender(t, "https://viset.vn", &RenderOptions{Template: TemplateCompact})
	if got := decode(t, img.RGBA); got != "https://viset.vn" {
		t.Fatalf("giải mã ra %q", got)
	}
}

func TestOutputFormats(t *testing.T) {
	img := mustRender(t, testQR, nil)
	dir := t.TempDir()
	for _, tc := range []struct {
		format Format
		magic  string
		mime   string
	}{
		{FormatPNG, "\x89PNG", "image/png"},
		{FormatJPEG, "\xff\xd8", "image/jpeg"},
		{"jpg", "\xff\xd8", "image/jpeg"},
		{FormatSVG, "<svg", "image/svg+xml"},
	} {
		b, err := img.Bytes(tc.format)
		if err != nil || !bytes.HasPrefix(b, []byte(tc.magic)) {
			t.Fatalf("%s: bytes sai (%v)", tc.format, err)
		}
		b64, _ := img.Base64(tc.format)
		raw, _ := base64.StdEncoding.DecodeString(b64)
		if !bytes.HasPrefix(raw, []byte(tc.magic)) {
			t.Errorf("%s: base64 sai", tc.format)
		}
		url, _ := img.DataURL(tc.format)
		if !strings.HasPrefix(url, "data:"+tc.mime+";base64,") {
			t.Errorf("%s: data URL sai: %.40s", tc.format, url)
		}
		path := filepath.Join(dir, "qr."+string(tc.format))
		if err := img.Save(path); err != nil {
			t.Fatal(err)
		}
		if saved, _ := os.ReadFile(path); !bytes.HasPrefix(saved, []byte(tc.magic)) {
			t.Errorf("%s: file sai", tc.format)
		}
	}
}

func TestSVGOutput(t *testing.T) {
	for _, shape := range []ModuleShape{ModuleSquare, ModuleRounded, ModuleCircle, ModuleGapped} {
		img := mustRender(t, testQR, &RenderOptions{Style: Style{ModuleShape: shape, Gradient: GradientVertical}})
		svg, err := img.SVG()
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			XMLName xml.Name `xml:"svg"`
			Width   string   `xml:"width,attr"`
			Defs    struct {
				Linear []struct{} `xml:"linearGradient"`
			} `xml:"defs"`
			Images []struct{} `xml:"image"`
		}
		if err := xml.Unmarshal([]byte(svg), &doc); err != nil {
			t.Fatalf("%s: SVG không hợp lệ: %v", shape, err)
		}
		if doc.Width != "500" || len(doc.Images) != 1 || len(doc.Defs.Linear) != 1 {
			t.Errorf("%s: SVG thiếu thành phần: %+v", shape, doc)
		}
	}
	svg, err := RenderSVG(testQR, &RenderOptions{NoCenterLogo: true, Style: Style{Transparent: true}})
	if err != nil || strings.Contains(svg, "<image") || strings.Contains(svg, "<rect") {
		t.Errorf("SVG không logo, nền trong suốt: %v", err)
	}
	if _, err := mustRender(t, testQR, &RenderOptions{Template: TemplateCompact}).SVG(); err == nil {
		t.Error("mẫu thẻ không được xuất SVG")
	}
}

func TestInvalidArguments(t *testing.T) {
	for name, opts := range map[string]*RenderOptions{
		"template": {Template: "poster"},
		"size":     {Size: 10},
		"scale":    {Template: TemplateCompact, Scale: -1},
		"module":   {Style: Style{ModuleShape: "star"}},
		"eye":      {Style: Style{EyeShape: "star"}},
		"gradient": {Style: Style{Gradient: "spiral"}},
		"ec":       {Style: Style{ErrorCorrection: "X"}},
	} {
		if _, err := Render(testQR, opts); err == nil {
			t.Errorf("%s: mong đợi lỗi", name)
		}
	}
	if _, err := mustRender(t, testQR, nil).Bytes("gif"); err == nil {
		t.Error("gif phải lỗi")
	}
	if _, err := ParseHex("#12"); err == nil {
		t.Error("màu sai phải lỗi")
	}
	if c := MustHex("#abc"); c != (color.RGBA{0xAA, 0xBB, 0xCC, 255}) {
		t.Errorf("MustHex = %v", c)
	}
}

func TestFormatAmount(t *testing.T) {
	for in, want := range map[int64]string{2000: "2.000 VND", 1234567: "1.234.567 VND", 5: "5 VND", -1000: "-1.000 VND"} {
		if got := FormatAmount(in); got != want {
			t.Errorf("FormatAmount(%d) = %q", in, got)
		}
	}
}
