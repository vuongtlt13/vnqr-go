package vnqr

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/image/vector"
)

const (
	kappa     = 0.5522847498307936 // điều khiển Bézier xấp xỉ cung 1/4 tròn
	bigRadius = 1e9                // bán kính "vô cùng" -> bị kẹp thành elip
)

// qrMatrix trả ma trận module (true = tối), không có quiet zone.
func qrMatrix(content string, level qrcode.RecoveryLevel) ([][]bool, error) {
	q, err := qrcode.New(content, level)
	if err != nil {
		return nil, fmt.Errorf("vnqr: không mã hoá được QR: %w", err)
	}
	q.DisableBorder = true
	return q.Bitmap(), nil
}

func finderOrigins(n int) [3][2]int { return [3][2]int{{0, 0}, {0, n - 7}, {n - 7, 0}} }

func inFinder(n, r, c int) bool {
	return (r < 7 && (c < 7 || c >= n-7)) || (r >= n-7 && c < 7)
}

// hole là vùng chừa cho logo giữa, theo đơn vị module.
type hole struct{ x0, y0, x1, y1 float64 }

func newHole(n int, ratio float64) *hole {
	if ratio <= 0 {
		return nil
	}
	side := float64(n) * ratio
	start := (float64(n) - side) / 2
	return &hole{start, start, start + side, start + side}
}

// forEachModule duyệt module dữ liệu tối (bỏ ô định vị và vùng logo) kèm cờ bo 4 góc
// (trái-trên, phải-trên, phải-dưới, trái-dưới) khi góc đó không có module kề.
func forEachModule(matrix [][]bool, h *hole, fn func(r, c int, corners [4]bool)) {
	n := len(matrix)
	dark := func(r, c int) bool {
		return r >= 0 && r < n && c >= 0 && c < n && matrix[r][c] && !inFinder(n, r, c)
	}
	for r := range n {
		for c := range n {
			if !dark(r, c) {
				continue
			}
			fr, fc := float64(r), float64(c)
			if h != nil && fc < h.x1 && fc+1 > h.x0 && fr < h.y1 && fr+1 > h.y0 {
				continue
			}
			up, down, left, right := dark(r-1, c), dark(r+1, c), dark(r, c-1), dark(r, c+1)
			fn(r, c, [4]bool{!(up || left), !(up || right), !(down || right), !(down || left)})
		}
	}
}

// addRRect thêm hình chữ nhật bo góc (bán kính từng góc tl, tr, br, bl; bị kẹp về
// nửa cạnh, nên bán kính lớn cho ra elip). Mọi hình cùng chiều để phép hợp liền mạch.
func addRRect(z *vector.Rasterizer, x0, y0, x1, y1 float64, radii [4]float64) {
	w, h := x1-x0, y1-y0
	if w <= 0 || h <= 0 {
		return
	}
	var rx, ry [4]float64
	for i, r := range radii {
		rx[i], ry[i] = math.Min(math.Max(r, 0), w/2), math.Min(math.Max(r, 0), h/2)
	}
	f := func(v float64) float32 { return float32(v) }
	k := kappa
	z.MoveTo(f(x0+rx[0]), f(y0))
	z.LineTo(f(x1-rx[1]), f(y0))
	if rx[1] > 0 {
		z.CubeTo(f(x1-rx[1]+rx[1]*k), f(y0), f(x1), f(y0+ry[1]-ry[1]*k), f(x1), f(y0+ry[1]))
	}
	z.LineTo(f(x1), f(y1-ry[2]))
	if rx[2] > 0 {
		z.CubeTo(f(x1), f(y1-ry[2]+ry[2]*k), f(x1-rx[2]+rx[2]*k), f(y1), f(x1-rx[2]), f(y1))
	}
	z.LineTo(f(x0+rx[3]), f(y1))
	if rx[3] > 0 {
		z.CubeTo(f(x0+rx[3]-rx[3]*k), f(y1), f(x0), f(y1-ry[3]+ry[3]*k), f(x0), f(y1-ry[3]))
	}
	z.LineTo(f(x0), f(y0+ry[0]))
	if rx[0] > 0 {
		z.CubeTo(f(x0), f(y0+ry[0]-ry[0]*k), f(x0+rx[0]-rx[0]*k), f(y0), f(x0+rx[0]), f(y0))
	}
	z.ClosePath()
}

func uniformRadii(r float64) [4]float64 { return [4]float64{r, r, r, r} }

// drawMasks vẽ mặt nạ alpha (khử răng cưa) cho module dữ liệu và ô định vị.
func drawMasks(matrix [][]bool, px int, st *resolvedStyle, h *hole) (data, eye *image.Alpha) {
	n := len(matrix)
	m := float64(px) / float64(n)
	snap := func(v float64) float64 { return math.Round(v * m) }
	bounds := image.Rect(0, 0, px, px)

	var z vector.Rasterizer
	z.Reset(px, px)
	forEachModule(matrix, h, func(r, c int, corners [4]bool) {
		fr, fc := float64(r), float64(c)
		switch st.module {
		case ModuleSquare:
			addRRect(&z, snap(fc), snap(fr), snap(fc+1), snap(fr+1), [4]float64{})
		case ModuleGapped:
			addRRect(&z, snap(fc+0.1), snap(fr+0.1), snap(fc+0.9), snap(fr+0.9), [4]float64{})
		case ModuleCircle:
			addRRect(&z, snap(fc+0.05), snap(fr+0.05), snap(fc+0.95), snap(fr+0.95), uniformRadii(bigRadius))
		default: // rounded: bo góc nào không có module kề -> các module liền khối
			var radii [4]float64
			for i, round := range corners {
				if round {
					radii[i] = bigRadius
				}
			}
			addRRect(&z, snap(fc), snap(fr), snap(fc+1), snap(fr+1), radii)
		}
	})
	data = image.NewAlpha(bounds)
	z.Draw(data, bounds, image.Opaque, image.Point{})

	// Ô định vị = vòng ngoài 7x7 - vòng giữa 5x5 + lõi 3x3.
	var levels [3]*image.Alpha
	for i, inset := range []float64{0, 1, 2} {
		z.Reset(px, px)
		radius := 0.0
		switch st.eye {
		case EyeCircle:
			radius = bigRadius
		case EyeRounded:
			radius = (2.0 - inset*0.6) * m
		}
		for _, o := range finderOrigins(n) {
			r0, c0 := float64(o[0]), float64(o[1])
			addRRect(&z, snap(c0+inset), snap(r0+inset), snap(c0+7-inset), snap(r0+7-inset), uniformRadii(radius))
		}
		levels[i] = image.NewAlpha(bounds)
		z.Draw(levels[i], bounds, image.Opaque, image.Point{})
	}
	eye = image.NewAlpha(bounds)
	for i := range eye.Pix {
		v := int(levels[0].Pix[i]) - int(levels[1].Pix[i]) + int(levels[2].Pix[i])
		eye.Pix[i] = uint8(min(max(v, 0), 255))
	}
	return data, eye
}

// paint trả nguồn màu (đơn sắc hoặc gradient) kích thước px x px.
func paint(px int, st *resolvedStyle) image.Image {
	if st.gradient == GradientNone {
		return image.NewUniform(st.color)
	}
	img := image.NewRGBA(image.Rect(0, 0, px, px))
	d := float64(max(px-1, 1))
	half := float64(px) / 2
	a, b := st.color, st.gradColor
	lerp := func(x, y uint8, t float64) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	for y := range px {
		row := img.Pix[y*img.Stride:]
		for x := range px {
			var t float64
			switch st.gradient {
			case GradientVertical:
				t = float64(y) / d
			case GradientHorizontal:
				t = float64(x) / d
			case GradientDiagonal:
				t = float64(x+y) / (2 * d)
			case GradientRadial:
				t = math.Min(1, math.Hypot(float64(x)+0.5-half, float64(y)+0.5-half)/half)
			}
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = lerp(a.R, b.R, t), lerp(a.G, b.G, t), lerp(a.B, b.B, t), 255
		}
	}
	return img
}

// fillRoundedRect tô hình chữ nhật bo góc khử răng cưa.
func fillRoundedRect(dst *image.RGBA, r image.Rectangle, radius float64, c color.Color) {
	var z vector.Rasterizer
	z.Reset(r.Dx(), r.Dy())
	addRRect(&z, 0, 0, float64(r.Dx()), float64(r.Dy()), uniformRadii(radius))
	z.Draw(dst, r, image.NewUniform(c), image.Point{})
}

func fillRect(dst *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Over)
}

// qrTile vẽ mã QR vuông px x px (không quiet zone) kèm logo giữa, nền theo style.
func qrTile(content string, px int, st *resolvedStyle, logo *image.RGBA, boxRatio, radiusRatio, iconRatio float64) (*image.RGBA, error) {
	matrix, err := qrMatrix(content, st.level)
	if err != nil {
		return nil, err
	}
	n := len(matrix)
	boxPx := 0
	if logo != nil {
		boxPx = int(float64(px) * boxRatio)
	}
	var h *hole
	if boxPx > 0 {
		h = newHole(n, float64(boxPx)/float64(px))
	}
	dataMask, eyeMask := drawMasks(matrix, px, st, h)

	tile := image.NewRGBA(image.Rect(0, 0, px, px))
	if !st.transparent {
		draw.Draw(tile, tile.Bounds(), image.NewUniform(st.bg), image.Point{}, draw.Src)
	}
	fill := paint(px, st)
	draw.DrawMask(tile, tile.Bounds(), fill, image.Point{}, dataMask, image.Point{}, draw.Over)
	eyeFill := fill
	if st.eyeColor != nil {
		eyeFill = image.NewUniform(*st.eyeColor)
	}
	draw.DrawMask(tile, tile.Bounds(), eyeFill, image.Point{}, eyeMask, image.Point{}, draw.Over)

	if logo != nil && boxPx > 0 {
		off := (px - boxPx) / 2
		boxColor := st.bg
		if st.transparent {
			boxColor = color.RGBA{255, 255, 255, 255}
		}
		fillRoundedRect(tile, image.Rect(off, off, off+boxPx, off+boxPx),
			math.Max(4, float64(boxPx)*radiusRatio), boxColor)
		iconPx := max(1, int(float64(boxPx)*iconRatio))
		icon := contain(logo, iconPx, iconPx)
		paste(tile, icon, (px-icon.Rect.Dx())/2, (px-icon.Rect.Dy())/2)
	}
	return tile, nil
}
