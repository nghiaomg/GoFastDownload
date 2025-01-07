package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"github.com/schollz/progressbar/v3"
)

const (
	chunkSize = 1024 * 1024
)

func main() {
	fmt.Println("\n=== CHƯƠNG TRÌNH TẢI FILE ĐA LUỒNG ===")
	fmt.Println("----------------------------------------")

	fmt.Print("\n→ Nhập URL file cần tải: ")
	var url string
	fmt.Scanln(&url)

	fmt.Print("\n⌛ Đang kiểm tra thông tin file...")
	
	resp, err := http.Head(url)
	if err != nil {
		fmt.Printf("\n❌ Lỗi: Không thể kết nối tới URL (%v)\n", err)
		return
	}
	defer resp.Body.Close()

	fileName := getFileNameFromURL(url, resp.Header.Get("Content-Type"))
	contentLength, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		fmt.Printf("\n❌ Lỗi: Không thể xác định kích thước file (%v)\n", err)
		return
	}

	fmt.Printf("\n✨ Thông tin file:")
	fmt.Printf("\n   • Tên file: %s", fileName)
	fmt.Printf("\n   • Kích thước: %.2f MB\n", float64(contentLength)/1024/1024)

	fmt.Print("\n→ Bạn có muốn tải file này? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("\n✖ Đã hủy tải xuống")
		return
	}

	outFile, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("\n❌ Lỗi: Không thể tạo file (%v)\n", err)
		return
	}
	defer outFile.Close()

	var wg sync.WaitGroup
	numParts := (contentLength + chunkSize - 1) / chunkSize
	partResults := make([][]byte, numParts)
	mutex := &sync.Mutex{}
	
	bar := progressbar.NewOptions(numParts,
		progressbar.OptionSetDescription("📥 Đang tải xuống..."),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for i := 0; i < numParts; i++ {
		wg.Add(1)
		go func(part int) {
			defer wg.Done()
			start := part * chunkSize
			end := start + chunkSize - 1
			if end > contentLength-1 {
				end = contentLength - 1
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Printf("\n❌ Lỗi phần %d: %v\n", part, err)
				return
			}

			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("\n❌ Lỗi tải phần %d: %v\n", part, err)
				return
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("\n❌ Lỗi đọc dữ liệu phần %d: %v\n", part, err)
				return
			}

			mutex.Lock()
			partResults[part] = data
				bar.Add(1)
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	fmt.Print("\n💾 Đang lưu file...")
	for _, part := range partResults {
		if _, err := outFile.Write(part); err != nil {
			fmt.Printf("\n❌ Lỗi: Không thể ghi file (%v)\n", err)
			return
		}
	}

	fmt.Printf("\n\n✅ Tải file thành công!\n")
	fmt.Printf("   📂 Đã lưu tại: %s\n", fileName)
	fmt.Println("----------------------------------------\n")
}

func getFileNameFromURL(url string, contentType string) string {
	baseName := path.Base(url)

	if !strings.Contains(baseName, ".") {
		ext := getExtensionFromContentType(contentType)
		if ext != "" {
			baseName += ext
		}
	}

	return baseName
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "application/pdf":
		return ".pdf"
	case "text/html":
		return ".html"
	case "application/vnd.android.package-archive":
		return ".apk"
	case "application/x-msdownload", "application/x-executable":
		return ".exe"
	case "application/x-rar-compressed", "application/vnd.rar":
		return ".rar"
	case "application/zip":
		return ".zip"
	case "application/x-7z-compressed":
		return ".7z"
	default:
		return ""
	}
}
