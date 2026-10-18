// Package avwiki は av-wiki.net をスクレイピングして各プラットフォームのコンテンツIDを取得します。
package avwiki

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const searchURL = "https://av-wiki.net/"

// Result はスクレイピング結果を表します。
type Result struct {
	Status           string `json:"status"`
	ErrorMessage     string `json:"error_message"`
	ManufacturerCode string `json:"manufacturer_code"`
	FanzaCode        string `json:"fanza_code"`
	MgsCode          string `json:"mgs_code"`
	SokmilCode       string `json:"sokmil_code"`
}

// Client は av-wiki.net スクレイパーのクライアントです。
type Client struct {
	httpClient *http.Client
}

// NewClient は Client を生成して返します。
func NewClient() *Client {
	return &Client{httpClient: http.DefaultClient}
}

// Search は keyword で av-wiki.net を検索し、各プラットフォームのコードを取得します。
func (c *Client) Search(keyword string) (*Result, error) {
	u, err := url.Parse(searchURL)
	if err != nil {
		return failedResult(fmt.Sprintf("url parse error: %v", err)), err
	}
	q := u.Query()
	q.Set("s", keyword)
	q.Set("post_type", "product")
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return failedResult(fmt.Sprintf("http request error: %v", err)), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("http status %d", resp.StatusCode)
		return failedResult(msg), fmt.Errorf("%s", msg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return failedResult(fmt.Sprintf("read body error: %v", err)), err
	}

	return ParseSearchResult(string(body))
}

// ParseSearchResult は検索結果HTMLを解析し、Result を返します。
func ParseSearchResult(htmlContent string) (*Result, error) {
	if strings.Contains(htmlContent, "検索ワードに一致する記事は見つかりませんでした") {
		return failedResult("no search results"), nil
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return failedResult(fmt.Sprintf("html parse error: %v", err)), err
	}

	result := &Result{Status: "succeeded"}

	mainNode := findFirstElement(doc, "main")
	if mainNode == nil {
		return failedResult("no main element found"), nil
	}

	article := findFirstElement(mainNode, "article")
	if article == nil {
		return failedResult("no article found in search results"), nil
	}

	result.ManufacturerCode = extractManufacturerCode(article)
	result.FanzaCode = extractCodeFromLink(article, "fanza")
	result.MgsCode = extractCodeFromLink(article, "mgs")
	result.SokmilCode = extractCodeFromLink(article, "sokmil")

	return result, nil
}

func failedResult(msg string) *Result {
	return &Result{Status: "failed", ErrorMessage: msg}
}

// findFirstElement は n 配下から指定タグの最初の要素を深さ優先で探索します。
// <main> → <article> のように探索起点を絞り込むことで、不要な領域の走査を避けます。
func findFirstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractManufacturerCode(n *html.Node) string {
	// <li><i class="fa fa-circle-o"></i>NHDTB-096</li> のパターンを探す
	var code string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if code != "" {
			return
		}
		if node.Type == html.ElementNode && node.Data == "i" && hasClass(node, "fa-circle-o") {
			if next := node.NextSibling; next != nil && next.Type == html.TextNode {
				code = strings.TrimSpace(next.Data)
			}
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return code
}

func extractCodeFromLink(n *html.Node, platform string) string {
	var code string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if code != "" {
			return
		}
		if node.Type == html.ElementNode && node.Data == "a" {
			linkText := getTextContent(node)
			href := getAttr(node, "href")
			if href == "" {
				goto children
			}
			switch platform {
			case "fanza":
				if strings.EqualFold(strings.TrimSpace(linkText), "FANZA") {
					code = extractFanzaCode(href)
					return
				}
			case "mgs":
				if strings.EqualFold(strings.TrimSpace(linkText), "MGS") {
					code = extractMgsCode(href)
					return
				}
			case "sokmil":
				if strings.EqualFold(strings.TrimSpace(linkText), "SOKMIL") {
					code = extractSokmilCode(href)
					return
				}
			}
		}
	children:
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return code
}

// extractFanzaCode は FANZA リンクURLから content ID を抽出します。
// URL例: https://al.fanza.co.jp/?lurl=https%3A%2F%2Fvideo.dmm.co.jp%2Fav%2Fcontent%2F%3Fid%3D1nhdtb00096%2F&...
func extractFanzaCode(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	lurl := u.Query().Get("lurl")
	if lurl == "" {
		return ""
	}
	inner, err := url.Parse(lurl)
	if err != nil {
		return ""
	}
	id := inner.Query().Get("id")
	return strings.TrimSuffix(id, "/")
}

// extractMgsCode は MGS リンクURLからプロダクトコードを抽出します。
// URL例: https://www.mgstage.com/product/product_detail/116NHDTB-096/?...
func extractMgsCode(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 3 {
		return ""
	}
	// path: product/product_detail/116NHDTB-096
	return parts[len(parts)-1]
}

// extractSokmilCode は SOKMIL リンクURLからアイテムコードを抽出します。
// URL例: https://www.sokmil.com/av/_item/item282537.htm?...
func extractSokmilCode(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	for _, p := range parts {
		if strings.HasPrefix(p, "item") && strings.HasSuffix(p, ".htm") {
			return strings.TrimSuffix(p, ".htm")
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getTextContent(c))
	}
	return sb.String()
}
