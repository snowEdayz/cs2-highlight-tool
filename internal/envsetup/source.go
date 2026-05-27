package envsetup

import (
	"fmt"
	"net/http"
	"strings"

	"cs2-highlight-tool-v2/internal/endpoints"
)

var preferredReleaseSourceFn = endpoints.PreferredReleaseSource

// SourceDetectionResult represents unified release-source status.
type SourceDetectionResult struct {
	Source      DownloadSource
	CountryCode string
	Message     string
}

// resolveDownloadSource decides source by GeoIP each startup.
// The update source is always github; country code is used by component download strategy.
func resolveDownloadSource(_ *http.Client) (SourceDetectionResult, error) {
	source, countryCode, err := preferredReleaseSourceFn()
	source = strings.ToLower(strings.TrimSpace(source))
	resolved := normalizeDownloadSource(source)
	if err != nil {
		return SourceDetectionResult{
			Source:      normalizeDownloadSource("github"),
			CountryCode: "",
			Message:     "GeoIP 检测失败，组件下载默认使用 URL 与 MIRROR 链路",
		}, nil
	}

	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if strings.EqualFold(countryCode, "CN") {
		return SourceDetectionResult{
			Source:      resolved,
			CountryCode: countryCode,
			Message:     "检测到中国区域，组件下载将优先使用 MIRROR 链路",
		}, nil
	}

	label := strings.ToUpper(string(resolved))
	if countryCode == "" {
		return SourceDetectionResult{
			Source:      resolved,
			CountryCode: "",
			Message:     "未检测到国家代码，组件下载将默认使用 URL 与 MIRROR 链路",
		}, nil
	}
	return SourceDetectionResult{
		Source:      resolved,
		CountryCode: countryCode,
		Message:     fmt.Sprintf("检测到国家 %s，组件下载将使用 %s 官方链路", countryCode, label),
	}, nil
}
