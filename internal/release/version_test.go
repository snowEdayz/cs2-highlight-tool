package release

import "testing"

func TestCurrentAppVersion_PrefersVersionField(t *testing.T) {
	value := CurrentAppVersion([]byte(`{"version":"1.2.3","info":{"productVersion":"9.9.9"}}`))
	if value != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", value)
	}
}

func TestCurrentAppVersion_UsesInfoProductVersion(t *testing.T) {
	value := CurrentAppVersion([]byte(`{"info":{"productVersion":"2.3.4"}}`))
	if value != "2.3.4" {
		t.Fatalf("version = %q, want 2.3.4", value)
	}
}

func TestCurrentAppVersion_Fallback(t *testing.T) {
	value := CurrentAppVersion([]byte(`{"name":"x"}`))
	if value != "0.0.0" {
		t.Fatalf("version = %q, want 0.0.0", value)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    int
	}{
		{"1.0.6", "1.0.10", -1},
		{"v1.2.0", "1.2", 0},
		{"1.3.0", "1.2.9", 1},
		{"", "0.0.1", -1},
	}
	for _, tt := range tests {
		got := CompareVersions(tt.current, tt.latest)
		if got != tt.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.want)
		}
	}
}
