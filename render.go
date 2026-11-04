package vnqr

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// RenderOptions là tuỳ chọn vẽ ảnh. Giá trị zero (hoặc nil) = mẫu qr_only 500px,
// có icon VietQR ở giữa.
type RenderOptions struct {
	// Template: TemplateQROnly (mặc định), TemplateCompact hoặc TemplatePrint.
	Template Template
	// Size: cạnh ảnh cho qr_only, mặc định 500, tối thiểu 50.
	Size int
	// Scale: hệ số phóng cho compact / print (2 -> 1080x1080, nét hơn khi in).
	// Mặc định 1.
	Scale float64
	// Style: kiểu vẽ module / màu sắc.
	Style Style
	// NoCenterLogo: không vẽ logo giữa.
	NoCenterLogo bool
	// CenterLogo: logo giữa riêng; nil = icon VietQR đóng gói sẵn.
	CenterLogo image.Image
	// CenterLogoRatio: tỉ lệ ô logo so với mã QR cho qr_only (0.10-0.28), mặc định 0.18.
	CenterLogoRatio float64
	// Bank: ghi đè ngân hàng hiển thị (BIN / mã / tên); mặc định đọc từ chuỗi VietQR.
	Bank string
	// BankLogo: logo ngân hàng riêng; mặc định dùng logo đóng gói theo BIN.
	BankLogo image.Image
	// AccountNo, AccountName, Description: ghi đè thông tin in ở mẫu print (mặc định
	// đọc từ chuỗi VietQR; tên chủ TK lấy từ tag 59 nếu có). Description có thể là
	// bản có dấu để hiển thị đẹp hơn.
	AccountNo   string
	AccountName string
	Description string
	// Amount: ghi đè số tiền in ở mẫu print; nil = đọc từ chuỗi, trỏ tới 0 = ẩn dòng.
	Amount *int64
	// AccentColor: màu khung, vạch ngăn và chữ của mẫu thẻ; nil = Accent.
	AccentColor color.Color
}

func (o *RenderOptions) normalize() (RenderOptions, error) {
	var r RenderOptions
	if o != nil {
		r = *o
	}
	if r.Template == "" {
		r.Template = TemplateQROnly
	}
	if r.Size == 0 {
		r.Size = 500
	}
	if r.Scale == 0 {
		r.Scale = 1
	}
	if r.CenterLogoRatio == 0 {
		r.CenterLogoRatio = 0.18
	}
	if r.Scale < 0 || math.IsInf(r.Scale, 0) || math.IsNaN(r.Scale) {
		return r, errors.New("vnqr: Scale phải > 0")
	}
	return r, nil
}

func (o *RenderOptions) centerLogo() *image.RGBA {
	if o.NoCenterLogo {
		return nil
	}
	if o.CenterLogo != nil {
		return trim(o.CenterLogo)
	}
	return assetImage("vietqr_icon.png")
}

func clampLogoRatio(r float64) float64 { return math.Min(math.Max(r, 0.10), 0.28) }

// Format là định dạng ảnh xuất ra.
type Format string

const (
	FormatPNG  Format = "png"
	FormatJPEG Format = "jpeg"
	FormatSVG  Format = "svg" // chỉ mẫu qr_only
)

func parseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimPrefix(s, ".")) {
	case "png", "":
		return FormatPNG, nil
	case "jpg", "jpeg":
		return FormatJPEG, nil
	case "svg":
		return FormatSVG, nil
	}
	return "", fmt.Errorf("vnqr: định dạng %q không hỗ trợ (png, jpeg, svg)", s)
}

// Image là ảnh đã vẽ kèm các hàm xuất.
type Image struct {
	// RGBA là ảnh raster. Nền trong suốt khi Style.Transparent ở mẫu qr_only.
	RGBA *image.RGBA
	svg  func() (string, error)
}

// Bytes mã hoá ảnh theo định dạng (png, jpeg/jpg, svg). JPEG được trải nền trắng.
func (img *Image) Bytes(format Format) ([]byte, error) {
	f, err := parseFormat(string(format))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	switch f {
	case FormatSVG:
		s, err := img.SVG()
		return []byte(s), err
	case FormatJPEG:
		flat := image.NewRGBA(img.RGBA.Rect)
		draw.Draw(flat, flat.Rect, image.White, image.Point{}, draw.Src)
		draw.Draw(flat, flat.Rect, img.RGBA, img.RGBA.Rect.Min, draw.Over)
		err = jpeg.Encode(&buf, flat, &jpeg.Options{Quality: 95})
	default:
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img.RGBA)
	}
	return buf.Bytes(), err
}

// Base64 trả ảnh mã hoá base64.
func (img *Image) Base64(format Format) (string, error) {
	b, err := img.Bytes(format)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// DataURL trả "data:image/...;base64,..." để nhúng HTML.
func (img *Image) DataURL(format Format) (string, error) {
	f, err := parseFormat(string(format))
	if err != nil {
		return "", err
	}
	mime := map[Format]string{FormatPNG: "image/png", FormatJPEG: "image/jpeg", FormatSVG: "image/svg+xml"}[f]
	b64, err := img.Base64(f)
	if err != nil {
		return "", err
	}
	return "data:" + mime + ";base64," + b64, nil
}

// SVG trả ảnh vector; chỉ hỗ trợ mẫu qr_only.
func (img *Image) SVG() (string, error) {
	if img.svg == nil {
		return "", errors.New("vnqr: SVG chỉ hỗ trợ mẫu qr_only")
	}
	return img.svg()
}

// Save lưu file, định dạng lấy theo đuôi (.png / .jpg / .jpeg / .svg).
func (img *Image) Save(path string) error {
	f, err := parseFormat(filepath.Ext(path))
	if err != nil {
		return err
	}
	b, err := img.Bytes(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Render vẽ ảnh mã QR cho chuỗi content (thường là chuỗi VietQR). opts = nil dùng
// mặc định: mẫu qr_only 500px có icon VietQR ở giữa.
func Render(content string, opts *RenderOptions) (*Image, error) {
	o, err := opts.normalize()
	if err != nil {
		return nil, err
	}
	st, err := o.Style.resolve()
	if err != nil {
		return nil, err
	}
	logo := o.centerLogo()

	switch o.Template {
	case TemplateQROnly:
		if o.Size < 50 {
			return nil, errors.New("vnqr: Size tối thiểu 50px")
		}
		padding := max(10, int(float64(o.Size)*0.02))
		tile, err := qrTile(content, o.Size-2*padding, st, logo, clampLogoRatio(o.CenterLogoRatio), 0.12, 0.82)
		if err != nil {
			return nil, err
		}
		canvas := image.NewRGBA(image.Rect(0, 0, o.Size, o.Size))
		if !st.transparent {
			draw.Draw(canvas, canvas.Rect, image.NewUniform(st.bg), image.Point{}, draw.Src)
		}
		paste(canvas, tile, padding, padding)
		svgOpts := o
		return &Image{RGBA: canvas, svg: func() (string, error) { return RenderSVG(content, &svgOpts) }}, nil
	case TemplateCompact, TemplatePrint:
		info := cardInfoFor(content, &o)
		canvas, err := renderCard(content, o.Template == TemplatePrint, o.Scale, st, logo, info, &o)
		if err != nil {
			return nil, err
		}
		return &Image{RGBA: canvas}, nil
	}
	return nil, fmt.Errorf("vnqr: mẫu %q không hỗ trợ (qr_only, compact, print)", o.Template)
}

// FormatAmount định dạng số tiền: 2000000 -> "2.000.000 VND".
func FormatAmount(amount int64) string {
	s := strconv.FormatInt(amount, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	return b.String() + " VND"
}

type cardInfo struct {
	bank        *Bank
	accountNo   string
	accountName string
	amount      int64
	description string
	cardService bool
}

func cardInfoFor(content string, o *RenderOptions) cardInfo {
	var info cardInfo
	if p, err := ParseLenient(content); err == nil {
		if b, ok := p.Bank(); ok {
			info.bank = &b
		}
		info.accountNo = p.Account
		info.accountName = p.MerchantName
		info.amount = p.Amount
		info.description = p.Content
		info.cardService = p.Service == ServiceCard
	}
	if o.Bank != "" {
		if b, ok := FindBank(o.Bank); ok {
			info.bank = &b
		}
	}
	if o.AccountNo != "" {
		info.accountNo = o.AccountNo
	}
	if o.AccountName != "" {
		info.accountName = o.AccountName
	}
	if o.Amount != nil {
		info.amount = *o.Amount
	}
	if o.Description != "" {
		info.description = o.Description
	}
	return info
}

type textLine struct {
	text string
	bold bool
	size float64
}

func printLines(info cardInfo) []textLine {
	var lines []textLine
	if info.amount > 0 {
		lines = append(lines, textLine{"Số tiền: " + FormatAmount(info.amount), false, 16})
	}
	if d := strings.TrimSpace(info.description); d != "" {
		lines = append(lines, textLine{"Nội dung CK: " + d, false, 16})
	}
	if n := strings.TrimSpace(info.accountName); n != "" {
		lines = append(lines, textLine{"Tên chủ TK: " + strings.ToUpper(n), false, 16})
	}
	if a := strings.TrimSpace(info.accountNo); a != "" {
		label := "Số TK"
		if info.cardService {
			label = "Số thẻ"
		}
		lines = append(lines, textLine{label + ": " + a, true, 18})
	}
	if info.bank != nil && info.bank.Name != "" {
		lines = append(lines, textLine{info.bank.Name, false, 15})
	}
	return lines
}

func textWidth(face font.Face, s string) float64 {
	return float64(font.MeasureString(face, s)) / 64
}

// fitText thu nhỏ font (tối đa ~30%), quá nữa thì cắt bớt và thêm dấu "…".
func fitText(text string, size float64, bold bool, maxWidth float64) (string, font.Face, error) {
	minSize := size * 0.7
	face, err := newFace(size, bold)
	if err != nil {
		return "", nil, err
	}
	for textWidth(face, text) > maxWidth && size > minSize {
		size -= 0.5
		if face, err = newFace(size, bold); err != nil {
			return "", nil, err
		}
	}
	if textWidth(face, text) > maxWidth {
		runes := []rune(text)
		for len(runes) > 0 && textWidth(face, string(runes)+"…") > maxWidth {
			runes = runes[:len(runes)-1]
		}
		text = strings.TrimRight(string(runes), " ") + "…"
	}
	return text, face, nil
}

func drawText(dst *image.RGBA, face font.Face, s string, x, baseline float64, c color.Color) {
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(math.Round(x * 64)), Y: fixed.Int26_6(math.Round(baseline * 64))},
	}
	d.DrawString(s)
}

func renderCard(content string, printable bool, scale float64, st *resolvedStyle, logo *image.RGBA, info cardInfo, o *RenderOptions) (*image.RGBA, error) {
	s := func(v float64) int { return int(math.RoundToEven(v * scale)) }
	accent := opaque(o.AccentColor, Accent)

	var lines []textLine
	baseW, baseH := 540.0, 540.0
	if printable {
		lines = printLines(info)
		baseW, baseH = 600, float64(590+len(lines)*28+16)
	}
	width, height := s(baseW), s(baseH)
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := st.bg
	if st.transparent {
		bg = color.RGBA{255, 255, 255, 255}
	}
	draw.Draw(canvas, canvas.Rect, image.NewUniform(bg), image.Point{}, draw.Src)

	pick := func(p, c float64) float64 {
		if printable {
			return p
		}
		return c
	}

	if header := assetImage("vietqr.png"); header != nil {
		header = contain(header, 10_000, s(pick(48, 46)))
		paste(canvas, header, (width-header.Rect.Dx())/2, s(pick(24, 18)))
	}

	const frame = 374.0
	frameX, frameY := (baseW-frame)/2, pick(96, 84)
	stroke := max(1, int(math.RoundToEven(2.5*scale)))
	x0, y0, x1, y1 := s(frameX), s(frameY), s(frameX+frame), s(frameY+frame)
	for _, r := range []image.Rectangle{
		image.Rect(x0, y0, x1, y0+stroke), image.Rect(x0, y1-stroke, x1, y1),
		image.Rect(x0, y0, x0+stroke, y1), image.Rect(x1-stroke, y0, x1, y1),
	} {
		fillRect(canvas, r, accent)
	}
	tile, err := qrTile(content, s(frame-18), st, logo, 58.0/356, 5.0/58, 46.0/58)
	if err != nil {
		return nil, err
	}
	paste(canvas, tile, s(frameX+9), s(frameY+9))

	// Footer: Napas247 | logo ngân hàng
	footerY := s(pick(516, 495))
	napas := assetImage("napas247.png")
	if napas != nil {
		napas = contain(napas, 10_000, s(35))
	}
	var bankImg *image.RGBA
	if o.BankLogo != nil {
		bankImg = trim(o.BankLogo)
	} else {
		bankImg = bankLogo(info.bank)
	}
	if bankImg != nil {
		bankImg = contain(bankImg, s(220), s(35))
	}
	fallback := ""
	if bankImg == nil && info.bank != nil {
		fallback = info.bank.ShortName
	}
	fallbackFace, err := newFace(float64(s(24)), true)
	if err != nil {
		return nil, err
	}
	bankW := 0
	switch {
	case bankImg != nil:
		bankW = bankImg.Rect.Dx()
	case fallback != "":
		bankW = int(math.RoundToEven(textWidth(fallbackFace, fallback)))
	}
	napasW := 0
	if napas != nil {
		napasW = napas.Rect.Dx()
	}
	gap, lineW := s(14), max(1, s(2))
	var x int
	if bankW > 0 && napasW > 0 {
		x = (width - (napasW + gap + lineW + gap + bankW)) / 2
		paste(canvas, napas, x, footerY-napas.Rect.Dy()/2)
		sepX := x + napasW + gap
		fillRect(canvas, image.Rect(sepX, footerY-s(18), sepX+lineW, footerY+s(18)+1), accent)
		x = sepX + lineW + gap
	} else {
		x = (width - napasW - bankW) / 2
		if napas != nil {
			paste(canvas, napas, x, footerY-napas.Rect.Dy()/2)
			x += napasW
		}
	}
	if bankImg != nil {
		paste(canvas, bankImg, x, footerY-bankImg.Rect.Dy()/2)
	} else if fallback != "" {
		m := fallbackFace.Metrics()
		baseline := float64(footerY) + float64(m.Ascent-m.Descent)/64/2
		drawText(canvas, fallbackFace, fallback, float64(x), baseline, bankText)
	}

	y := float64(s(566))
	for _, l := range lines {
		text, face, err := fitText(l.text, l.size*scale, l.bold, float64(width-s(40)))
		if err != nil {
			return nil, err
		}
		drawText(canvas, face, text, float64(width)/2-textWidth(face, text)/2, y, accent)
		if l.bold {
			y += 29 * scale
		} else {
			y += 26 * scale
		}
	}
	return canvas, nil
}
