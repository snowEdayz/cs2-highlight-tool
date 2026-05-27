package release

import (
	"bytes"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	AdPlacementMainStepsTopBanner = "main_steps_top_banner"
)

type AdsManifest struct {
	Version   string   `json:"version"`
	UpdatedAt string   `json:"updated_at"`
	Items     []AdItem `json:"items"`
}

type AdItem struct {
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	Placement string `json:"placement"`
	ClickURL  string `json:"click_url"`
	Sponsor   string `json:"sponsor"`
	Title     string `json:"title"`
	RichHTML  string `json:"rich_html"`
	ImageURL  string `json:"image_url"`
	ImageAlt  string `json:"image_alt,omitempty"`
}

type manifestAdsPayload struct {
	Version   string              `json:"version"`
	UpdatedAt string              `json:"updated_at"`
	Items     []manifestAdPayload `json:"items"`
}

type manifestAdPayload struct {
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	Placement string `json:"placement"`
	ClickURL  string `json:"click_url"`
	Sponsor   string `json:"sponsor"`
	Title     string `json:"title"`
	RichHTML  string `json:"rich_html"`
	ImageURL  string `json:"image_url"`
	ImageAlt  string `json:"image_alt"`
}

var (
	allowedRichHTMLTags = map[string]struct{}{
		"p":      {},
		"br":     {},
		"strong": {},
		"b":      {},
		"em":     {},
		"i":      {},
		"u":      {},
		"s":      {},
		"ul":     {},
		"ol":     {},
		"li":     {},
		"a":      {},
		"code":   {},
		"pre":    {},
	}
	dropWholeRichHTMLTags = map[string]struct{}{
		"script":   {},
		"style":    {},
		"iframe":   {},
		"object":   {},
		"embed":    {},
		"svg":      {},
		"math":     {},
		"textarea": {},
		"noscript": {},
		"meta":     {},
		"link":     {},
	}
)

func parseManifestAds(payload manifestAdsPayload) (AdsManifest, []string) {
	out := AdsManifest{
		Version:   strings.TrimSpace(payload.Version),
		UpdatedAt: strings.TrimSpace(payload.UpdatedAt),
		Items:     make([]AdItem, 0, len(payload.Items)),
	}
	errors := make([]string, 0)
	for i, raw := range payload.Items {
		item, ok, reason := validateAndNormalizeAd(raw)
		if !ok {
			if strings.TrimSpace(reason) != "" {
				errors = append(errors, reasonWithIndex(i, raw.ID, reason))
			}
			continue
		}
		out.Items = append(out.Items, item)
	}
	return out, errors
}

func validateAndNormalizeAd(raw manifestAdPayload) (AdItem, bool, string) {
	if !raw.Enabled {
		return AdItem{}, false, ""
	}
	id := strings.TrimSpace(raw.ID)
	if id == "" {
		return AdItem{}, false, "missing id"
	}
	placement := strings.TrimSpace(raw.Placement)
	if placement != AdPlacementMainStepsTopBanner {
		return AdItem{}, false, "unsupported placement"
	}
	clickURL, ok := normalizeExternalLinkURL(raw.ClickURL)
	if !ok {
		return AdItem{}, false, "invalid click_url"
	}
	sponsor := strings.TrimSpace(raw.Sponsor)
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return AdItem{}, false, "missing title"
	}
	richHTML := strings.TrimSpace(raw.RichHTML)
	if richHTML == "" {
		return AdItem{}, false, "missing rich_html"
	}
	sanitizedRichHTML, sanitizeOK := sanitizeRichHTML(richHTML)
	if !sanitizeOK {
		return AdItem{}, false, "invalid rich_html"
	}
	imageURL, ok := normalizeAdImageURL(raw.ImageURL)
	if !ok {
		return AdItem{}, false, "invalid image_url"
	}

	return AdItem{
		ID:        id,
		Enabled:   true,
		Placement: placement,
		ClickURL:  clickURL,
		Sponsor:   sponsor,
		Title:     title,
		RichHTML:  sanitizedRichHTML,
		ImageURL:  imageURL,
		ImageAlt:  strings.TrimSpace(raw.ImageAlt),
	}, true, ""
}

func reasonWithIndex(idx int, id, reason string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "ad[" + strconv.Itoa(idx) + "]: " + strings.TrimSpace(reason)
	}
	return "ad[" + strconv.Itoa(idx) + "](" + id + "): " + strings.TrimSpace(reason)
}

func normalizeExternalHTTPURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", false
	}
	return parsed.String(), true
}

func normalizeAdImageURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	switch scheme {
	case "http", "https":
		if strings.TrimSpace(parsed.Host) == "" {
			return "", false
		}
		return parsed.String(), true
	case "data":
		if isDataImageURL(parsed.Opaque) {
			return parsed.String(), true
		}
		return "", false
	default:
		return "", false
	}
}

func isDataImageURL(opaque string) bool {
	opaque = strings.TrimSpace(opaque)
	if opaque == "" || strings.ContainsAny(opaque, "\r\n") {
		return false
	}
	comma := strings.Index(opaque, ",")
	if comma <= 0 {
		return false
	}
	mediaPart := strings.ToLower(strings.TrimSpace(opaque[:comma]))
	return strings.HasPrefix(mediaPart, "image/")
}

func normalizeExternalLinkURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	switch scheme {
	case "http", "https":
		if strings.TrimSpace(parsed.Host) == "" {
			return "", false
		}
		return parsed.String(), true
	case "mailto":
		return normalizeMailtoURL(parsed)
	default:
		return "", false
	}
}

func normalizeMailtoURL(parsed *url.URL) (string, bool) {
	if parsed == nil {
		return "", false
	}
	target := strings.TrimSpace(parsed.Opaque)
	if target == "" {
		target = strings.TrimSpace(parsed.Host)
	}
	if target == "" {
		target = strings.TrimSpace(strings.TrimPrefix(parsed.Path, "/"))
	}
	if target == "" || strings.ContainsAny(target, "\r\n") {
		return "", false
	}

	addressList := target
	if idx := strings.Index(addressList, "?"); idx >= 0 {
		addressList = addressList[:idx]
	}
	addressList = strings.TrimSpace(addressList)
	if addressList == "" {
		return "", false
	}
	for _, part := range strings.Split(addressList, ",") {
		addr := strings.TrimSpace(part)
		if addr == "" {
			return "", false
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return "", false
		}
	}

	return parsed.String(), true
}

func sanitizeRichHTML(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	root := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(raw), root)
	if err != nil {
		return "", false
	}
	wrapper := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	for _, n := range nodes {
		wrapper.AppendChild(n)
	}
	sanitizeRichHTMLTree(wrapper)

	var buf bytes.Buffer
	for child := wrapper.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&buf, child); err != nil {
			return "", false
		}
	}
	sanitized := strings.TrimSpace(buf.String())
	if sanitized == "" {
		return "", false
	}
	return sanitized, true
}

func sanitizeRichHTMLTree(node *html.Node) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		switch child.Type {
		case html.CommentNode:
			detachNode(child)
		case html.ElementNode:
			tag := strings.ToLower(strings.TrimSpace(child.Data))
			if _, ok := dropWholeRichHTMLTags[tag]; ok {
				detachNode(child)
				child = next
				continue
			}
			sanitizeRichHTMLTree(child)
			if _, ok := allowedRichHTMLTags[tag]; !ok {
				unwrapNode(child)
				child = next
				continue
			}
			if !filterRichHTMLAttrs(child, tag) {
				child = next
				continue
			}
		default:
			sanitizeRichHTMLTree(child)
		}
		child = next
	}
}

func filterRichHTMLAttrs(node *html.Node, tag string) bool {
	filtered := make([]html.Attribute, 0, len(node.Attr))
	for _, attr := range node.Attr {
		key := strings.ToLower(strings.TrimSpace(attr.Key))
		val := strings.TrimSpace(attr.Val)
		if tag == "a" {
			switch key {
			case "href":
				normalized, ok := normalizeExternalLinkURL(val)
				if !ok {
					continue
				}
				filtered = append(filtered, html.Attribute{Key: "href", Val: normalized})
			case "target":
				if val == "_blank" {
					filtered = append(filtered, html.Attribute{Key: "target", Val: "_blank"})
				}
			case "rel":
				// overwrite rel at the end for external anchors.
			}
		}
	}
	if tag == "a" {
		hasHref := false
		for _, attr := range filtered {
			if attr.Key == "href" {
				hasHref = true
				break
			}
		}
		if !hasHref {
			unwrapNode(node)
			return false
		}
		filtered = append(filtered, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
	}
	node.Attr = filtered
	return true
}

func unwrapNode(node *html.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		node.RemoveChild(child)
		parent.InsertBefore(child, node)
		child = next
	}
	parent.RemoveChild(node)
}

func detachNode(node *html.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}
	parent.RemoveChild(node)
}
