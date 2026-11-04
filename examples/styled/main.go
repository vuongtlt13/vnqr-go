// Xuất các ảnh mẫu vào thư mục examples/styled:
//
//	go run ./examples/styled
package main

import (
	"log"
	"path/filepath"
	"runtime"

	vnqr "github.com/vuongtlt13/vnqr-go"
)

func main() {
	_, file, _, _ := runtime.Caller(0)
	out := filepath.Dir(file)

	qr, err := vnqr.Build(vnqr.Payment{
		Bank:         "MB",
		Account:      "0123456789",
		Amount:       2_000_000,
		Content:      "Thanh toán đơn DH123",
		MerchantName: "Nguyễn Văn A",
	})
	if err != nil {
		log.Fatal(err)
	}

	samples := map[string]*vnqr.RenderOptions{
		"compact.png": {Template: vnqr.TemplateCompact},
		"compact_rounded.png": {
			Template: vnqr.TemplateCompact,
			Style: vnqr.Style{
				ModuleShape: vnqr.ModuleRounded,
				EyeShape:    vnqr.EyeRounded,
				Color:       vnqr.MustHex("#003B70"),
			},
		},
		"dots_gradient.png": {
			Style: vnqr.Style{
				ModuleShape:   vnqr.ModuleCircle,
				EyeShape:      vnqr.EyeCircle,
				Gradient:      vnqr.GradientDiagonal,
				GradientColor: vnqr.MustHex("#7A1FA2"),
			},
		},
		"print.png": {
			Template:    vnqr.TemplatePrint,
			Description: "Thanh toán đơn DH123",
		},
		"transparent.svg": {
			Style: vnqr.Style{ModuleShape: vnqr.ModuleRounded, Transparent: true},
		},
	}
	for name, opts := range samples {
		img, err := vnqr.Render(qr, opts)
		if err != nil {
			log.Fatal(err)
		}
		if err := img.Save(filepath.Join(out, name)); err != nil {
			log.Fatal(err)
		}
		log.Println("đã lưu", name)
	}
}
