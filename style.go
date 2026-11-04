package vnqr

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Template là mẫu ảnh.
type Template string

const (
	// TemplateQROnly (mặc định): chỉ mã QR vuông Size x Size px, xuất được SVG.
	TemplateQROnly Template = "qr_only"
	// TemplateCompact: thẻ 540x540 (x Scale) có logo VietQR, Napas247 và ngân hàng.
	TemplateCompact Template = "compact"
	// TemplatePrint: thẻ 600px (x Scale) như compact, thêm số tiền, nội dung, chủ
	// TK, số TK và tên ngân hàng.
	TemplatePrint Template = "print"
)

// ModuleShape là hình của module dữ liệu.
type ModuleShape string

const (
	ModuleSquare  ModuleShape = "square"  // chuẩn (mặc định)
	ModuleRounded ModuleShape = "rounded" // bo góc liền khối
	ModuleCircle  ModuleShape = "circle"  // chấm tròn
	ModuleGapped  ModuleShape = "gapped"  // ô vuông có khe
)

// EyeShape là hình 3 ô định vị (finder) ở góc.
type EyeShape string

const (
	EyeSquare  EyeShape = "square"
	EyeRounded EyeShape = "rounded"
	EyeCircle  EyeShape = "circle"
)

// Gradient là kiểu đổ màu chuyển từ Style.Color sang Style.GradientColor.
type Gradient string

const (
	GradientNone       Gradient = ""
	GradientVertical   Gradient = "vertical"
	GradientHorizontal Gradient = "horizontal"
	GradientDiagonal   Gradient = "diagonal"
	GradientRadial     Gradient = "radial"
)

// ErrorCorrection là mức sửa lỗi của mã QR.
type ErrorCorrection string

const (
	ECLow      ErrorCorrection = "L"
	ECMedium   ErrorCorrection = "M"
	ECQuartile ErrorCorrection = "Q" // mặc định, chừa chỗ cho logo giữa
	ECHigh     ErrorCorrection = "H"
)

// Accent là màu khung, vạch ngăn và chữ mặc định của mẫu thẻ (#003B70).
var Accent = color.RGBA{0, 59, 112, 255}

var bankText = color.RGBA{0, 82, 155, 255}

// Style là kiểu vẽ mã QR. Giá trị zero là mã QR đen trên nền trắng, module vuông.
//
// Nên giữ màu module đậm trên nền sáng để mọi app ngân hàng quét được.
type Style struct {
	ModuleShape   ModuleShape
	EyeShape      EyeShape
	Color         color.Color // màu module, nil = đen
	Gradient      Gradient
	GradientColor color.Color // màu cuối gradient, nil = Accent
	EyeColor      color.Color // màu riêng cho ô định vị, nil = theo Color/gradient
	Background    color.Color // màu nền, nil = trắng
	// Transparent: nền trong suốt (PNG/SVG mẫu qr_only); bỏ qua Background.
	Transparent     bool
	ErrorCorrection ErrorCorrection // "" = Q
}

type resolvedStyle struct {
	module      ModuleShape
	eye         EyeShape
	color       color.RGBA
	gradient    Gradient
	gradColor   color.RGBA
	eyeColor    *color.RGBA
	bg          color.RGBA
	transparent bool
	level       qrcode.RecoveryLevel
}

func (s Style) resolve() (*resolvedStyle, error) {
	r := &resolvedStyle{
		module:      s.ModuleShape,
		eye:         s.EyeShape,
		color:       opaque(s.Color, color.RGBA{0, 0, 0, 255}),
		gradient:    s.Gradient,
		gradColor:   opaque(s.GradientColor, Accent),
		bg:          opaque(s.Background, color.RGBA{255, 255, 255, 255}),
		transparent: s.Transparent,
	}
	switch r.module {
	case "":
		r.module = ModuleSquare
	case ModuleSquare, ModuleRounded, ModuleCircle, ModuleGapped:
	default:
		return nil, fmt.Errorf("vnqr: ModuleShape %q không hỗ trợ (square, rounded, circle, gapped)", r.module)
	}
	switch r.eye {
	case "":
		r.eye = EyeSquare
	case EyeSquare, EyeRounded, EyeCircle:
	default:
		return nil, fmt.Errorf("vnqr: EyeShape %q không hỗ trợ (square, rounded, circle)", r.eye)
	}
	switch r.gradient {
	case GradientNone, GradientVertical, GradientHorizontal, GradientDiagonal, GradientRadial:
	default:
		return nil, fmt.Errorf("vnqr: Gradient %q không hỗ trợ (vertical, horizontal, diagonal, radial)", r.gradient)
	}
	switch s.ErrorCorrection {
	case ECLow:
		r.level = qrcode.Low
	case ECMedium:
		r.level = qrcode.Medium
	case ECQuartile, "":
		r.level = qrcode.High // go-qrcode: High = 25% (mức Q)
	case ECHigh:
		r.level = qrcode.Highest
	default:
		return nil, fmt.Errorf("vnqr: ErrorCorrection %q không hỗ trợ (L, M, Q, H)", s.ErrorCorrection)
	}
	if s.EyeColor != nil {
		c := opaque(s.EyeColor, color.RGBA{})
		r.eyeColor = &c
	}
	return r, nil
}

// opaque chuyển màu về RGB không trong suốt (bỏ kênh alpha), nil -> def.
func opaque(c color.Color, def color.RGBA) color.RGBA {
	if c == nil {
		return def
	}
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return color.RGBA{n.R, n.G, n.B, 255}
}

// ParseHex đọc màu dạng "#RRGGBB" hoặc "#RGB" (dấu # không bắt buộc).
func ParseHex(s string) (color.RGBA, error) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return color.RGBA{}, fmt.Errorf("vnqr: màu %q không hợp lệ", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("vnqr: màu %q không hợp lệ", s)
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, nil
}

// MustHex giống ParseHex nhưng panic khi sai định dạng; dùng cho hằng số.
func MustHex(s string) color.RGBA {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

func hexColor(c color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
