package vnqr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"strconv"
	"strings"
)

func svgNum(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	if s == "-0" {
		return "0"
	}
	return s
}

func svgRRect(b *strings.Builder, x, y, w, h float64, radii [4]float64) {
	tl, tr, br, bl := radii[0], radii[1], radii[2], radii[3]
	fmt.Fprintf(b, "M%s %sH%s", svgNum(x+tl), svgNum(y), svgNum(x+w-tr))
	if tr > 0 {
		fmt.Fprintf(b, "a%s %s 0 0 1 %s %s", svgNum(tr), svgNum(tr), svgNum(tr), svgNum(tr))
	}
	fmt.Fprintf(b, "V%s", svgNum(y+h-br))
	if br > 0 {
		fmt.Fprintf(b, "a%s %s 0 0 1 %s %s", svgNum(br), svgNum(br), svgNum(-br), svgNum(br))
	}
	fmt.Fprintf(b, "H%s", svgNum(x+bl))
	if bl > 0 {
		fmt.Fprintf(b, "a%s %s 0 0 1 %s %s", svgNum(bl), svgNum(bl), svgNum(-bl), svgNum(-bl))
	}
	fmt.Fprintf(b, "V%s", svgNum(y+tl))
	if tl > 0 {
		fmt.Fprintf(b, "a%s %s 0 0 1 %s %s", svgNum(tl), svgNum(tl), svgNum(tl), svgNum(-tl))
	}
	b.WriteByte('Z')
}

func svgGradient(st *resolvedStyle, id string) string {
	stops := fmt.Sprintf(`<stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/>`,
		hexColor(st.color), hexColor(st.gradColor))
	if st.gradient == GradientRadial {
		return fmt.Sprintf(`<radialGradient id="%s" cx=".5" cy=".5" r=".5">%s</radialGradient>`, id, stops)
	}
	x2, y2 := 0, 1
	switch st.gradient {
	case GradientHorizontal:
		x2, y2 = 1, 0
	case GradientDiagonal:
		x2, y2 = 1, 1
	}
	return fmt.Sprintf(`<linearGradient id="%s" x1="0" y1="0" x2="%d" y2="%d">%s</linearGradient>`, id, x2, y2, stops)
}

// RenderSVG vẽ mã QR dạng SVG (vector, mẫu qr_only). Dùng các trường Size, Style,
// NoCenterLogo, CenterLogo, CenterLogoRatio của opts (nil = mặc định); logo giữa
// được nhúng dạng PNG base64.
func RenderSVG(content string, opts *RenderOptions) (string, error) {
	o, err := opts.normalize()
	if err != nil {
		return "", err
	}
	if o.Size < 50 {
		return "", fmt.Errorf("vnqr: Size tối thiểu 50px")
	}
	st, err := o.Style.resolve()
	if err != nil {
		return "", err
	}
	matrix, err := qrMatrix(content, st.level)
	if err != nil {
		return "", err
	}
	n := len(matrix)
	nf := float64(n)
	size := o.Size
	paddingPx := max(10, int(float64(size)*0.02))
	pad := float64(paddingPx) * nf / float64(size-2*paddingPx)
	view := nf + 2*pad
	logo := o.centerLogo()
	var h *hole
	if logo != nil {
		h = newHole(n, clampLogoRatio(o.CenterLogoRatio))
	}

	var data strings.Builder
	forEachModule(matrix, h, func(r, c int, corners [4]bool) {
		fr, fc := float64(r), float64(c)
		switch st.module {
		case ModuleSquare:
			fmt.Fprintf(&data, "M%d %dh1v1h-1z", c, r)
		case ModuleGapped:
			fmt.Fprintf(&data, "M%s %sh.8v.8h-.8z", svgNum(fc+0.1), svgNum(fr+0.1))
		case ModuleCircle:
			fmt.Fprintf(&data, "M%s %sa.45 .45 0 1 0 .9 0a.45 .45 0 1 0 -.9 0z", svgNum(fc+0.05), svgNum(fr+0.5))
		default:
			var radii [4]float64
			for i, round := range corners {
				if round {
					radii[i] = 0.5
				}
			}
			svgRRect(&data, fc, fr, 1, 1, radii)
		}
	})

	var eyes strings.Builder
	for _, o := range finderOrigins(n) {
		for _, inset := range []float64{0, 1, 2} {
			side := 7 - 2*inset
			radius := 0.0
			switch st.eye {
			case EyeCircle:
				radius = side / 2
			case EyeRounded:
				radius = 2.0 - inset*0.6
			}
			svgRRect(&eyes, float64(o[1])+inset, float64(o[0])+inset, side, side, uniformRadii(radius))
		}
	}

	fill := hexColor(st.color)
	defs := ""
	if st.gradient != GradientNone {
		defs = "<defs>" + svgGradient(st, "vnqr-grad") + "</defs>"
		fill = "url(#vnqr-grad)"
	}
	eyeFill := fill
	if st.eyeColor != nil {
		eyeFill = hexColor(*st.eyeColor)
	}
	rendering := "auto"
	if st.module == ModuleSquare {
		rendering = "crispEdges"
	}

	var out strings.Builder
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="%s %s %s %s" shape-rendering="%s">`,
		size, size, svgNum(-pad), svgNum(-pad), svgNum(view), svgNum(view), rendering)
	out.WriteString(defs)
	if !st.transparent {
		fmt.Fprintf(&out, `<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`,
			svgNum(-pad), svgNum(-pad), svgNum(view), svgNum(view), hexColor(st.bg))
	}
	fmt.Fprintf(&out, `<path fill="%s" d="%s"/>`, fill, data.String())
	fmt.Fprintf(&out, `<path fill="%s" fill-rule="evenodd" d="%s"/>`, eyeFill, eyes.String())

	if logo != nil && h != nil {
		side := h.x1 - h.x0
		bg := "#FFFFFF"
		if !st.transparent {
			bg = hexColor(st.bg)
		}
		fmt.Fprintf(&out, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"/>`,
			svgNum(h.x0), svgNum(h.y0), svgNum(side), svgNum(side), svgNum(side*0.12), bg)
		icon := side * 0.82
		var buf bytes.Buffer
		if err := png.Encode(&buf, contain(logo, 256, 256)); err != nil {
			return "", err
		}
		fmt.Fprintf(&out, `<image x="%s" y="%s" width="%s" height="%s" preserveAspectRatio="xMidYMid meet" href="data:image/png;base64,%s"/>`,
			svgNum((nf-icon)/2), svgNum((nf-icon)/2), svgNum(icon), svgNum(icon),
			base64.StdEncoding.EncodeToString(buf.Bytes()))
	}
	out.WriteString("</svg>")
	return out.String(), nil
}
