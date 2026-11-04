package vnqr

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // đăng ký decoder JPEG cho LoadImage
	_ "image/png"
	"io"
	"math"
	"os"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// LoadImage đọc ảnh PNG / JPEG (dùng cho RenderOptions.CenterLogo / BankLogo).
func LoadImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("vnqr: không đọc được ảnh: %w", err)
	}
	return img, nil
}

// LoadImageFile đọc ảnh PNG / JPEG từ file.
func LoadImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadImage(f)
}

var assetCache sync.Map // tên -> *image.RGBA (nil nếu không có)

// assetImage trả ảnh đóng gói đã cắt viền trong suốt, nil nếu không có. Ảnh dùng
// chung giữa các lần gọi, chỉ được đọc.
func assetImage(name string) *image.RGBA {
	if v, ok := assetCache.Load(name); ok {
		return v.(*image.RGBA)
	}
	var img *image.RGBA
	if data, err := embedded.ReadFile("assets/" + name); err == nil {
		if decoded, _, err := image.Decode(bytes.NewReader(data)); err == nil {
			img = trim(decoded)
		}
	}
	v, _ := assetCache.LoadOrStore(name, img)
	return v.(*image.RGBA)
}

func bankLogo(b *Bank) *image.RGBA {
	if b == nil {
		return nil
	}
	return assetImage("logos/" + b.BIN + ".png")
}

// trim chuyển ảnh về RGBA gốc (0,0) và cắt viền trong suốt.
func trim(src image.Image) *image.RGBA {
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	w, h := rgba.Rect.Dx(), rgba.Rect.Dy()
	minX, minY, maxX, maxY := w, h, -1, -1
	for y := range h {
		row := rgba.Pix[y*rgba.Stride:]
		for x := range w {
			if row[x*4+3] != 0 {
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
			}
		}
	}
	if maxX < 0 || (minX == 0 && minY == 0 && maxX == w-1 && maxY == h-1) {
		return rgba
	}
	out := image.NewRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	draw.Draw(out, out.Bounds(), rgba, image.Pt(minX, minY), draw.Src)
	return out
}

// contain thu/phóng ảnh vừa khung width x height, giữ tỉ lệ.
func contain(src *image.RGBA, width, height int) *image.RGBA {
	sw, sh := float64(src.Rect.Dx()), float64(src.Rect.Dy())
	ratio := math.Min(float64(width)/sw, float64(height)/sh)
	w := max(1, int(math.RoundToEven(sw*ratio)))
	h := max(1, int(math.RoundToEven(sh*ratio)))
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

// paste vẽ src (gốc 0,0) đè lên dst tại (x, y).
func paste(dst *image.RGBA, src *image.RGBA, x, y int) {
	r := src.Bounds().Add(image.Pt(x, y))
	draw.Draw(dst, r, src, image.Point{}, draw.Over)
}

type fontPair struct{ regular, bold *opentype.Font }

var loadFonts = sync.OnceValues(func() (fontPair, error) {
	var fp fontPair
	for _, f := range []struct {
		name string
		dst  **opentype.Font
	}{
		{"BeVietnamPro-Regular.ttf", &fp.regular},
		{"BeVietnamPro-SemiBold.ttf", &fp.bold},
	} {
		data, err := embedded.ReadFile("assets/fonts/" + f.name)
		if err != nil {
			return fp, fmt.Errorf("vnqr: thiếu font %s: %w", f.name, err)
		}
		if *f.dst, err = opentype.Parse(data); err != nil {
			return fp, fmt.Errorf("vnqr: font %s hỏng: %w", f.name, err)
		}
	}
	return fp, nil
})

func newFace(size float64, bold bool) (font.Face, error) {
	fp, err := loadFonts()
	if err != nil {
		return nil, err
	}
	f := fp.regular
	if bold {
		f = fp.bold
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    max(1, math.RoundToEven(size)),
		DPI:     72,
		Hinting: font.HintingNone,
	})
}
