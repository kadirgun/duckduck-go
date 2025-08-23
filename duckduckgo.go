package duckduckgo

import (
	"fmt"
	"regexp"
	"time"

	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

type DuckDuckGo struct {
	client *resty.Client
}

func New() *DuckDuckGo {
	client := resty.New()
	client.SetHeader("User-Agent", "PostmanRuntime/7.45.0")
	client.SetHeader("Accept", "*/*")
	client.SetHeader("Accept-Language", "en-US,en;q=0.9")
	client.SetHeader("Accept-Encoding", "gzip, deflate")
	client.SetHeader("Connection", "keep-alive")
	client.SetHeader("Cache-Control", "no-cache")

	return &DuckDuckGo{
		client: client,
	}
}

func (d *DuckDuckGo) Search(query string, count int) ([]SearchResult, error) {
	scriptLink, err := d.getScriptLink(query)
	if err != nil {
		return nil, err
	}

	fmt.Println("Script Link:", scriptLink)

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
	req := d.client.R()
	req.SetHeader("Referer", "https://duckduckgo.com/")
	req.SetHeader("Host", "links.duckduckgo.com")

	resp, err := req.Get(scriptLink)
	if err != nil {
		return nil, err
	}

	js := resp.String()

	re := regexp.MustCompile(`DDG\.pageLayout\.load\('d',(\[.*?\])\)`)
	match := re.FindStringSubmatch(js)
	if len(match) < 2 {
		return nil, fmt.Errorf("no results found")
	}

	arrayStr := match[1]

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
			Title:       gjson.Get(item.String(), "t").String(),
			Description: gjson.Get(item.String(), "a").String(),
			Link:        gjson.Get(item.String(), "u").String(),
		})
	}

	return result, nil
}

func (d *DuckDuckGo) getScriptLink(query string) (string, error) {
	req := d.client.R()
	req.SetQueryParam("q", query)
	req.SetQueryParam("t", "h_")
	req.SetQueryParam("ia", "web")

	res, err := req.Get("https://duckduckgo.com")

	if err != nil {
		return "", err
	}

	if res.StatusCode() != 200 {
		return "", fmt.Errorf("unexpected status code: %d", res.StatusCode())
	}

	content := res.String()
	re := regexp.MustCompile(`<link id="deep_preload_link" rel="preload" as="script" href="(https?://[^\s/$.?#].[^\s]*)">`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("script link not found")
	}

	return matches[1], nil
}
