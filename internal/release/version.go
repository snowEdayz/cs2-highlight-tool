package release

import (
	"encoding/json"
	"strconv"
	"strings"
)

func CurrentAppVersion(wailsConfigData []byte) string {
	if len(wailsConfigData) == 0 {
		return "0.0.0"
	}
	var data struct {
		Version string `json:"version"`
		Info    struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsConfigData, &data); err != nil {
		return "0.0.0"
	}
	if strings.TrimSpace(data.Version) != "" {
		return strings.TrimSpace(data.Version)
	}
	if strings.TrimSpace(data.Info.ProductVersion) != "" {
		return strings.TrimSpace(data.Info.ProductVersion)
	}
	return "0.0.0"
}

func CompareVersions(current, latest string) int {
	a := parseVersion(current)
	b := parseVersion(latest)
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	for len(a) < max {
		a = append(a, 0)
	}
	for len(b) < max {
		b = append(b, 0)
	}
	for i := 0; i < max; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) []int {
	v = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
	if v == "" {
		return []int{0}
	}
	main := strings.FieldsFunc(v, func(r rune) bool {
		return r == '-' || r == '+'
	})[0]
	parts := strings.Split(main, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
