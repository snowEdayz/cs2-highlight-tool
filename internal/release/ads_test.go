package release

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchUnifiedLatest_ParsesAndFiltersAds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"project":{"id":"cs2-highlight-tool-v2","name":"cs2-highlight-tool-v2"},
			"dependencies":{
				"advancedfx":{
					"name":"advancedfx",
					"repo":"advancedfx/advancedfx",
					"latest_tag":"v2.189.9",
					"latest":{"tag_name":"v2.189.9","assets":[{"name":"hlae.zip","url":"https://dl.example.com/hlae.zip"}]}
				}
			},
			"ads":{
				"version":"2.0",
				"updated_at":"2026-05-08T10:00:00Z",
				"items":[
					{
						"id":"valid_card",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"https://ad.example.com/landing",
						"sponsor":"",
						"title":"Sponsored title",
						"rich_html":"<p>hello <strong>world</strong></p>",
						"image_url":"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='256' height='144'><rect width='100%25' height='100%25' fill='%230b3b2e'/></svg>",
						"image_alt":"img alt"
					},
					{
						"id":"invalid_placement",
						"enabled":true,
						"placement":"import_methods_card",
						"click_url":"https://ad.example.com/x",
						"sponsor":"Acme",
						"title":"bad",
						"rich_html":"<p>bad</p>",
						"image_url":"https://cdn.example.com/x.jpg"
					},
					{
						"id":"invalid_click_url",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"javascript:alert(1)",
						"sponsor":"Acme",
						"title":"bad",
						"rich_html":"<p>bad</p>",
						"image_url":"https://cdn.example.com/y.jpg"
					},
					{
						"id":"invalid_image_url",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"https://ad.example.com/html",
						"sponsor":"Acme",
						"title":"bad",
						"rich_html":"<p>bad</p>",
						"image_url":"javascript:alert(2)"
					},
					{
						"id":"invalid_data_image_url",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"https://ad.example.com/html2",
						"sponsor":"Acme",
						"title":"bad",
						"rich_html":"<p>bad</p>",
						"image_url":"data:text/html,<p>bad</p>"
					},
					{
						"id":"legacy_shape_ignored",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"https://ad.example.com/legacy",
						"content":{"image_url":"https://cdn.example.com/legacy.jpg"}
					},
					{
						"id":"second_valid_card",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"https://ad.example.com/second",
						"sponsor":"Beta",
						"title":"Second title",
						"rich_html":"<p>second line</p>",
						"image_url":"https://cdn.example.com/b.jpg"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	latest, err := FetchUnifiedLatest(server.URL)
	if err != nil {
		t.Fatalf("FetchUnifiedLatest error: %v", err)
	}
	if latest.Ads.Version != "2.0" {
		t.Fatalf("ads version = %q", latest.Ads.Version)
	}
	if len(latest.Ads.Items) != 2 {
		t.Fatalf("ads item count = %d, want 2, errors=%v", len(latest.Ads.Items), latest.AdValidationErrors)
	}

	if latest.Ads.Items[0].ID != "valid_card" {
		t.Fatalf("first ad id = %q", latest.Ads.Items[0].ID)
	}
	if latest.Ads.Items[1].ID != "second_valid_card" {
		t.Fatalf("second ad id = %q", latest.Ads.Items[1].ID)
	}
	if latest.Ads.Items[0].RichHTML != "<p>hello <strong>world</strong></p>" {
		t.Fatalf("rich_html mismatch: %q", latest.Ads.Items[0].RichHTML)
	}
	if latest.Ads.Items[0].Sponsor != "" {
		t.Fatalf("first ad sponsor = %q, want empty", latest.Ads.Items[0].Sponsor)
	}
}

func TestFetchUnifiedLatest_SanitizesRichHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"project":{"id":"cs2-highlight-tool-v2","name":"cs2-highlight-tool-v2"},
			"dependencies":{
				"advancedfx":{"name":"advancedfx","repo":"advancedfx/advancedfx","latest":{"assets":[{"name":"hlae.zip","url":"https://dl.example.com/hlae.zip"}]}}
			},
			"ads":{
				"version":"2.0",
				"items":[
					{
						"id":"sanitized_card",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"https://ad.example.com/card",
						"sponsor":"Sanitizer",
						"title":"Safe Content",
						"rich_html":"<p onclick='x()'>safe</p><script>alert(1)</script><a href='javascript:alert(2)'>x</a><a href='https://safe.example/path' target='_blank' onclick='x()'>ok</a>",
						"image_url":"https://cdn.example.com/card.jpg"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	latest, err := FetchUnifiedLatest(server.URL)
	if err != nil {
		t.Fatalf("FetchUnifiedLatest error: %v", err)
	}
	if len(latest.Ads.Items) != 1 {
		t.Fatalf("ads item count = %d, want 1, errors=%v", len(latest.Ads.Items), latest.AdValidationErrors)
	}
	got := latest.Ads.Items[0].RichHTML
	want := "<p>safe</p>x<a href=\"https://safe.example/path\" target=\"_blank\" rel=\"noopener noreferrer\">ok</a>"
	if got != want {
		t.Fatalf("sanitized rich_html = %q, want %q", got, want)
	}
}

func TestFetchUnifiedLatest_AllowsMailtoForClickAndRichHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"project":{"id":"cs2-highlight-tool-v2","name":"cs2-highlight-tool-v2"},
			"dependencies":{
				"advancedfx":{"name":"advancedfx","repo":"advancedfx/advancedfx","latest":{"assets":[{"name":"hlae.zip","url":"https://dl.example.com/hlae.zip"}]}}
			},
			"ads":{
				"version":"2.0",
				"items":[
					{
						"id":"mailto_card",
						"enabled":true,
						"placement":"main_steps_top_banner",
						"click_url":"mailto:hk_snow@yeah.net",
						"sponsor":"",
						"title":"Mail Contact",
						"rich_html":"<a href='mailto:hk_snow@yeah.net' target='_blank'>联系我们</a>",
						"image_url":"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='256' height='144'><rect width='100%25' height='100%25' fill='%230b3b2e'/></svg>"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	latest, err := FetchUnifiedLatest(server.URL)
	if err != nil {
		t.Fatalf("FetchUnifiedLatest error: %v", err)
	}
	if len(latest.Ads.Items) != 1 {
		t.Fatalf("ads item count = %d, want 1, errors=%v", len(latest.Ads.Items), latest.AdValidationErrors)
	}
	ad := latest.Ads.Items[0]
	if ad.ClickURL != "mailto:hk_snow@yeah.net" {
		t.Fatalf("click_url = %q", ad.ClickURL)
	}
	if ad.Sponsor != "" {
		t.Fatalf("sponsor = %q, want empty", ad.Sponsor)
	}
	wantRich := "<a href=\"mailto:hk_snow@yeah.net\" target=\"_blank\" rel=\"noopener noreferrer\">联系我们</a>"
	if ad.RichHTML != wantRich {
		t.Fatalf("rich_html = %q, want %q", ad.RichHTML, wantRich)
	}
}
