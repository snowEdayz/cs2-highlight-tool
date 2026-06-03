package envsetup

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cs2-highlight-tool-v2/internal/download"
	"cs2-highlight-tool-v2/internal/release"
)

func TestCancelStartupDownloadIgnoresInactiveOrUnsupportedComponent(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	state := svc.CancelStartupDownload(componentHLAE)
	if got := findStepStatusForTest(state, componentHLAE); got != statusPending {
		t.Fatalf("inactive HLAE status = %s, want %s", got, statusPending)
	}

	state = svc.CancelStartupDownload(componentCS2)
	if got := findStepStatusForTest(state, componentCS2); got != statusPending {
		t.Fatalf("unsupported CS2 status = %s, want %s", got, statusPending)
	}
}

func findStepStatusForTest(state StartupState, componentID string) string {
	for _, step := range state.Steps {
		if step.ID == componentID {
			return step.Status
		}
	}
	return ""
}

func TestDownloadAndInstallWithFallbackCancelStopsRemainingCandidates(t *testing.T) {
	firstStarted := make(chan struct{})
	secondHit := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first.zip":
			_, _ = w.Write([]byte("partial"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			close(firstStarted)
			<-r.Context().Done()
		case "/second.zip":
			secondHit <- struct{}{}
			_, _ = w.Write([]byte("second"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)
	candidates := []releaseAssetCandidate{
		{
			Source:   DownloadSourceGitHub,
			Asset:    release.Asset{Name: "first.zip"},
			AssetURL: server.URL + "/first.zip",
			URLKind:  urlKindDirect,
		},
		{
			Source:   DownloadSourceGitHub,
			Asset:    release.Asset{Name: "second.zip"},
			AssetURL: server.URL + "/second.zip",
			URLKind:  urlKindMirror,
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.downloadAndInstallWithFallback(componentHLAE, "v2.0.0", candidates, func(path string) error {
			return nil
		})
	}()

	<-firstStarted
	svc.CancelStartupDownload(componentHLAE)
	err := <-errCh
	if !errors.Is(err, download.ErrCanceled) {
		t.Fatalf("downloadAndInstallWithFallback error = %v, want ErrCanceled", err)
	}
	select {
	case <-secondHit:
		t.Fatal("second candidate was attempted after cancel")
	default:
	}
}
