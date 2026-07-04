package duckduckgo

import (
	"fmt"
	"html"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/tidwall/gjson"
)

type DuckDuckGo struct {
	client        tls_client.HttpClient
	proxy         string
	timeout       time.Duration
	userAgent     string
	retryCount    int
	proxyFallback bool
	directClient  tls_client.HttpClient
}

var preloadRe = regexp.MustCompile(`(?s)<link[^>]*\bid="deep_preload_link"[^>]*\bhref="(https?://[^"]+)"`)

func New(opts ...Option) *DuckDuckGo {
	d := &DuckDuckGo{
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(d)
	}

	headers := http.Header{}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	if d.userAgent != "" {
		ua = d.userAgent
	}
	headers.Set("User-Agent", ua)
	headers.Set("Accept", "*/*")
	headers.Set("Accept-Language", "en-US,en;q=0.9")
	headers.Set("Accept-Encoding", "gzip, deflate, br")
	headers.Set("Connection", "keep-alive")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Sec-Ch-Ua", `"Chromium";v="142", "Google Chrome";v="142", "Not_A Brand";v="99"`)

	tlsOptions := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_146_PSK),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithDefaultHeaders(headers),
		tls_client.WithTimeoutSeconds(int(d.timeout.Seconds())),
	}

	if d.proxy != "" {
		tlsOptions = append(tlsOptions, tls_client.WithProxyUrl(d.proxy))
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), tlsOptions...)
	if err != nil {
		panic(err)
	}

	dd := &DuckDuckGo{
		client:        client,
		proxy:         d.proxy,
		timeout:       d.timeout,
		userAgent:     d.userAgent,
		retryCount:    d.retryCount,
		proxyFallback: d.proxyFallback,
	}

	if d.proxy != "" && d.proxyFallback {
		directOpts := []tls_client.HttpClientOption{
			tls_client.WithClientProfile(profiles.Chrome_146_PSK),
			tls_client.WithNotFollowRedirects(),
			tls_client.WithDefaultHeaders(headers),
			tls_client.WithTimeoutSeconds(int(d.timeout.Seconds())),
		}
		directClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), directOpts...)
		if err == nil {
			dd.directClient = directClient
		}
	}

	return dd
}

func (d *DuckDuckGo) Search(query string, count int) ([]SearchResult, error) {
	scriptLink, err := d.retryGetScriptLink(query)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0)

	for scriptLink != "" {
		script, err := d.getResults(scriptLink)

		if err != nil {
			break
		}

		results = append(results, script.Items...)

		if len(results) >= count || len(script.Items) < 10 {
			break
		}

		scriptLink = script.Next
		time.Sleep(2 * time.Second)
	}

	return results[:min(count, len(results))], nil
}

type ScriptResult struct {
	Items []SearchResult
	Next  string
}

func (d *DuckDuckGo) getResults(scriptLink string) (*ScriptResult, error) {
	req, err := http.NewRequest("GET", scriptLink, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	jsBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	js := string(jsBytes)

	arrayStr := extractJSONArray(js, `DDG.pageLayout.load('d',`)
	if arrayStr == "" {
		return nil, fmt.Errorf("no results found")
	}

	items := gjson.Parse(arrayStr).Array()
	if len(items) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	result := &ScriptResult{
		Items: []SearchResult{},
	}

	for _, item := range items {
		nextPage := gjson.Get(item.String(), "n")
		if nextPage.Exists() {
			result.Next = nextPage.String()
			result.Next = fmt.Sprintf("https://links.duckduckgo.com%s", result.Next)
			continue
		}

		result.Items = append(result.Items, SearchResult{
			Title:       html.UnescapeString(gjson.Get(item.String(), "t").String()),
			Description: html.UnescapeString(gjson.Get(item.String(), "a").String()),
			Link:        gjson.Get(item.String(), "u").String(),
		})
	}

	return result, nil
}

func (d *DuckDuckGo) retryGetScriptLink(query string) (string, error) {
	var lastErr error

	for i := range d.retryCount + 1 {
		if i > 0 {
			time.Sleep(time.Second)
		}

		link, err := d.getScriptLink(query)
		if err == nil {
			return link, nil
		}

		lastErr = err
	}

	// proxy fallback: son denemede de başarısız olduysa direkt bağlantı dene
	if d.proxy != "" && d.proxyFallback && d.directClient != nil {
		link, err := d.getScriptLinkWithClient(query, d.directClient)
		if err == nil {
			return link, nil
		}
	}

	return "", lastErr
}

func (d *DuckDuckGo) getScriptLink(query string) (string, error) {
	return d.getScriptLinkWithClient(query, d.client)
}

func (d *DuckDuckGo) getScriptLinkWithClient(query string, client tls_client.HttpClient) (string, error) {
	urlValues := url.Values{
		"q":  {query},
		"t":  {"h_"},
		"ia": {"web"},
	}

	reqUrl := url.URL{
		Scheme:   "https",
		Host:     "duckduckgo.com",
		RawQuery: urlValues.Encode(),
	}

	fmt.Println("Request URL:", reqUrl.String())

	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)

	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	defer res.Body.Close()

	contentBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	content := string(contentBytes)

	link, err := extractScriptLink(content)
	if err != nil {
		return "", err
	}

	return link, nil
}

func extractScriptLink(content string) (string, error) {
	matches := preloadRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("script link not found")
	}

	return matches[1], nil
}

// extractJSONArray finds the first JSON array after the given marker in JS content
// using bracket counting instead of regex to handle nested structures correctly.
func extractJSONArray(js, marker string) string {
	idx := strings.Index(js, marker)
	if idx == -1 {
		return ""
	}

	start := idx + len(marker)
	if start >= len(js) || js[start] != '[' {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(js); i++ {
		ch := js[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			switch ch {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					return js[start : i+1]
				}
			}
		}
	}

	return ""
}
