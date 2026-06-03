package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ProgressFunc func(active bool, percent float64, indeterminate bool)

var ErrCanceled = errors.New("下载已取消")

func File(url, targetPath string, emitProgress ProgressFunc) error {
	return FileWithContext(context.Background(), url, targetPath, emitProgress)
}

func FileWithContext(ctx context.Context, url, targetPath string, emitProgress ProgressFunc) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	if emitProgress != nil {
		emitProgress(true, 0, true)
		defer emitProgress(false, 0, false)
	}

	client := &http.Client{Timeout: 3 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return ErrCanceled
		}
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
	defer func() {
		_ = out.Close()
		if errors.Is(err, ErrCanceled) {
			_ = os.Remove(targetPath)
		}
	}()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 64*1024)
	lastEmit := time.Now()
	for {
		select {
		case <-ctx.Done():
			return ErrCanceled
		default:
		}

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
			if errors.Is(readErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				return ErrCanceled
			}
			return readErr
		}
	}
	if emitProgress != nil {
		emitProgress(true, 100, false)
	}
	return nil
}
