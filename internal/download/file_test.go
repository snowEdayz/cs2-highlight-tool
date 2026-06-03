package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWithContextCancelRemovesPartialFile(t *testing.T) {
	chunkWritten := make(chan struct{})
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("partial"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		close(chunkWritten)
		<-releaseHandler
	}))
	defer server.Close()
	defer close(releaseHandler)

	targetPath := filepath.Join(t.TempDir(), "download.tmp")
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- FileWithContext(ctx, server.URL, targetPath, nil)
	}()

	<-chunkWritten
	cancel()
	err := <-errCh
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("FileWithContext error = %v, want ErrCanceled", err)
	}
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("partial target exists after cancel, stat err = %v", statErr)
	}
}
