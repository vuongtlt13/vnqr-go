# TUYÊN BỐ MIỄN TRỪ TRÁCH NHIỆM (DISCLAIMER)

## 1. Thương hiệu và tài liệu tiêu chuẩn

- **Nhãn hiệu NAPAS và VietQR:** Tên gọi, biểu trưng, logo và nhãn hiệu thương mại **VietQR**, **NAPAS** và **NAPAS247** thuộc quyền sở hữu của **Công ty Cổ phần Thanh toán Quốc gia Việt Nam (NAPAS)**. Logo, tên và nhãn hiệu của các ngân hàng, ví điện tử trong thư mục `assets/logos/` thuộc quyền sở hữu của từng tổ chức tương ứng.
- **Mục đích sử dụng tài nguyên hình ảnh:** Các logo được đóng gói chỉ để hiển thị mã VietQR đúng nhận diện khi thanh toán qua Dịch vụ chuyển nhanh NAPAS247. Thư viện không được NAPAS, Ngân hàng Nhà nước Việt Nam hay bất kỳ ngân hàng nào tài trợ, chứng nhận hoặc bảo trợ. Người dùng tự kiểm tra và tuân thủ quy định về nhận diện thương hiệu của NAPAS và ngân hàng trước khi in hoặc phát hành ảnh QR.
- **Tiêu chuẩn kỹ thuật:** Định dạng mã thanh toán trong thư viện dựa trên:
  1. *Tiêu chuẩn cơ sở cho mã phản hồi nhanh (QR Code) trong lĩnh vực thanh toán tại Việt Nam* do **Ngân hàng Nhà nước Việt Nam** ban hành.
  2. *Quy định về Định dạng mã VietQR trong Dịch vụ chuyển nhanh NAPAS247 (Phiên bản 1.0 - 09/2021)* do **NAPAS** công bố.
  3. *EMVCo QR Code Specification for Payment Systems (Merchant-Presented Mode)*.
- **Tôn trọng Điều khoản và Điều kiện (T&C) của NAPAS:** Thư viện không sở hữu, cũng không tuyên bố hay ngầm ý có quyền sở hữu đối với tài liệu đặc tả, nhãn hiệu, logo hay tài sản sở hữu trí tuệ nào của NAPAS, NHNN hoặc EMVCo. Thư viện chỉ dùng định dạng VietQR cho mục đích kỹ thuật: tạo và đọc mã thanh toán qua mạng lưới NAPAS247.

## 2. Bản quyền mã nguồn

- `vnqr-go` là thư viện mã nguồn mở, phát hành theo [giấy phép MIT](LICENSE). Đây là bản Go của [vnqr](https://github.com/vuongtlt13/vnqr) (Python).
- Bố cục mẫu ảnh `compact`, `print`, `qr_only` được chuyển thể từ [vietqr-core](https://github.com/Ngochiendev/vietqr-core) (giấy phép MIT).
- Font **Be Vietnam Pro** phát hành theo SIL Open Font License 1.1, xem `assets/fonts/OFL.txt`.
- Danh mục ngân hàng (`data/banks.json`) tổng hợp từ API công khai `https://api.vietqr.io/v2/banks`.
- Phần mềm được cung cấp **"NGUYÊN BẢN" (AS-IS)**, không kèm bất kỳ bảo đảm nào, dù rõ ràng hay ngụ ý, về tính sẵn sàng, tính liên tục, tính chính xác hoặc sự phù hợp cho một mục đích cụ thể.

## 3. Trách nhiệm vận hành và an toàn dữ liệu

Đơn vị hoặc cá nhân sử dụng, triển khai thư viện chịu hoàn toàn trách nhiệm về:

1. Tính chính xác và tính hợp pháp của thông tin thanh toán đưa vào mã QR: số tài khoản, mã BIN ngân hàng, tên người thụ hưởng, số tiền, nội dung chuyển khoản.
2. Kiểm tra mã QR quét được bằng app ngân hàng thật trước khi phát hành, nhất là khi đổi màu, đổi hình module hoặc chèn logo riêng (`Style`, `CenterLogo`).
3. Bảo mật dữ liệu khách hàng và tuân thủ pháp luật Việt Nam cùng các thông tư của Ngân hàng Nhà nước về an toàn, bảo mật thông tin trong thanh toán ngân hàng và thanh toán điện tử.

Tác giả và những người đóng góp không chịu trách nhiệm về bất kỳ khiếu nại, tranh chấp, tổn thất tài chính, giao dịch chuyển nhầm hay sự cố kỹ thuật nào phát sinh từ việc sử dụng, triển khai hoặc tích hợp thư viện vào hệ thống thực tế.
