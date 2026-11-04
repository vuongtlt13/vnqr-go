// Command crawl-banks tải danh sách ngân hàng mới nhất từ https://api.vietqr.io/v2/banks
// và ghi đè data/banks.json (danh mục đóng gói trong thư viện):
//
//	go run ./cmd/crawl-banks              # ghi data/banks.json
//	go run ./cmd/crawl-banks -o banks.json
//
// Logo ngân hàng mới (assets/logos/<BIN>.png) cần thêm thủ công.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const apiURL = "https://api.vietqr.io/v2/banks"

func main() {
	out := flag.String("o", "data/banks.json", "file đầu ra")
	flag.Parse()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		log.Fatalf("gọi API lỗi: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API trả HTTP %d", resp.StatusCode)
	}
	var body struct {
		Code string            `json:"code"`
		Desc string            `json:"desc"`
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		log.Fatalf("đọc JSON lỗi: %v", err)
	}
	if body.Code != "00" || len(body.Data) == 0 {
		log.Fatalf("API trả code %q: %s", body.Code, body.Desc)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(body.Data); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("đã lưu %d ngân hàng vào %s\n", len(body.Data), *out)
}
