package envsetup

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureReleaseSnapshot_UsesUnifiedAPI(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(unifiedPayloadForTest()))
	}))
	defer server.Close()

	t.Setenv("CS2_RELEASE_API_URL", server.URL)

	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	if err := svc.ensureReleaseSnapshot(defaultDownloadSource(), true); err != nil {
		t.Fatalf("ensureReleaseSnapshot(force) error: %v", err)
	}
	if err := svc.ensureReleaseSnapshot(defaultDownloadSource(), false); err != nil {
		t.Fatalf("ensureReleaseSnapshot(cache) error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
