// Package vnqr tạo, đọc và vẽ mã QR thanh toán VietQR (chuẩn EMVCo / NAPAS247).
//
// Tạo chuỗi VietQR rồi vẽ thành ảnh thẻ VietQR:
//
//	qr, err := vnqr.Build(vnqr.Payment{
//		Bank:    "MB", // BIN "970422", mã "MB" hoặc tên "MBBank"
//		Account: "0123456789",
//		Amount:  2_000_000,
//		Content: "Thanh toán đơn DH123", // tự bỏ dấu
//	})
//	if err != nil { ... }
//	img, err := vnqr.Render(qr, &vnqr.RenderOptions{Template: vnqr.TemplateCompact})
//	if err != nil { ... }
//	err = img.Save("vietqr.png")
//
// Giải mã: [Parse], [VerifyCRC]. Danh mục ngân hàng: [FindBank], [ResolveBIN],
// [ListBanks]. Vẽ ảnh: [Render], [RenderSVG], [Style].
//
// Bố cục mẫu ảnh compact / print / qr_only chuyển thể từ
// https://github.com/Ngochiendev/vietqr-core (MIT). Đây là bản Go của thư viện
// Python https://github.com/vuongtlt13/vnqr.
package vnqr
