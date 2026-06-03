package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ProgressFunc func(active bool, percent float64, indeterminate bool)

func File(url, targetPath string, emitProgress ProgressFunc) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	if emitProgress != nil {
		emitProgress(true, 0, true)
		defer emitProgress(false, 0, false)
	}

	client := &http.Client{Timeout: 3 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 64*1024)
	lastEmit := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			downloaded += int64(n)
			if total > 0 && time.Since(lastEmit) > 200*time.Millisecond {
				if emitProgress != nil {
					emitProgress(true, float64(downloaded)*100/float64(total), false)
				}
				lastEmit = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if emitProgress != nil {
		emitProgress(true, 100, false)
	}
	return nil
}
