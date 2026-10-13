package toolkit

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

const documentCSS = `body{margin:0;background:#f7f9f5;color:#293e37;font:16px/1.7 system-ui,sans-serif}article{max-width:900px;margin:32px auto;padding:32px;background:white;border-radius:16px;overflow-wrap:anywhere}h1,h2,h3{line-height:1.3}pre{padding:16px;background:#f3f5f2;border:1px solid #dce5dc;border-radius:8px;overflow:auto;white-space:pre;tab-size:2}code{font:0.88em/1.7 ui-monospace,monospace}p code,li code{padding:2px 5px;background:#edf3ed;border-radius:4px}table{border-collapse:collapse;max-width:100%;display:block;overflow:auto}td,th{border:1px solid #dce5dc;padding:10px}th{background:#edf3ed}blockquote{border-left:3px solid #93b4a6;margin-left:0;padding-left:18px}img{max-width:100%;height:auto}a{color:#075b53}details{padding:12px;border:1px solid #dce5dc;border-radius:8px}input{pointer-events:none}.image-placeholder{display:inline-block;padding:8px 12px;background:#edf3ed;border-radius:6px;color:#61716a}@media(max-width:600px){article{margin:0;padding:20px}}`

func renderMarkdown(values map[string]string) (Result, error) {
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.DefinitionList), goldmark.WithParserOptions(parser.WithAutoHeadingID()), goldmark.WithRendererOptions(goldhtml.WithUnsafe()))
	var rendered bytes.Buffer
	if err := markdown.Convert([]byte(values["input"]), &rendered); err != nil {
		return Result{}, err
	}
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("div", "details", "summary", "input")
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[a-zA-Z0-9_+-]+$`)).OnElements("code")
	policy.AllowAttrs("id").Matching(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)).OnElements("h1", "h2", "h3", "h4", "h5", "h6", "li")
	policy.AllowAttrs("align").Matching(regexp.MustCompile(`^(left|center|right)$`)).OnElements("div", "p", "th", "td")
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	policy.AllowAttrs("checked", "disabled").OnElements("input")
	policy.AllowAttrs("open").OnElements("details")
	safe := policy.Sanitize(rendered.String())
	var base *url.URL
	if strings.TrimSpace(values["base"]) != "" {
		parsed, err := url.Parse(strings.TrimSpace(values["base"]))
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return Result{}, errors.New("Use an HTTPS asset base URL without credentials, such as https://raw.githubusercontent.com/owner/repo/main/.")
		}
		base = parsed
	}
	root, err := html.Parse(strings.NewReader(safe))
	if err != nil {
		return Result{}, err
	}
	images := 0
	hidden := 0
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "img" {
			images++
			alt := "image"
			src := ""
			for _, attr := range node.Attr {
				if attr.Key == "alt" && attr.Val != "" {
					alt = attr.Val
				}
				if attr.Key == "src" {
					src = attr.Val
				}
			}
			resolved, parseErr := url.Parse(src)
			if parseErr == nil && base != nil {
				resolved = base.ResolveReference(resolved)
			}
			if values["images"] != "allow" || parseErr != nil || resolved.Scheme != "https" || resolved.Host == "" || resolved.User != nil {
				hidden++
				node.Data = "span"
				node.Attr = []html.Attribute{{Key: "class", Val: "image-placeholder"}}
				node.AppendChild(&html.Node{Type: html.TextNode, Data: "Image: " + alt})
			} else {
				for i := range node.Attr {
					if node.Attr[i].Key == "src" {
						node.Attr[i].Val = resolved.String()
					}
				}
			}
		}
		if node.Type == html.ElementNode && node.Data == "a" {
			for index, attribute := range node.Attr {
				if attribute.Key != "href" {
					continue
				}
				if strings.HasPrefix(attribute.Val, "#") {
					continue
				}
				parsed, err := url.Parse(attribute.Val)
				if err != nil {
					continue
				}
				if base != nil {
					parsed = base.ResolveReference(parsed)
					node.Attr[index].Val = parsed.String()
				}
				node.Attr = append(node.Attr, html.Attribute{Key: "target", Val: "_blank"}, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
				break
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	var body *html.Node
	var find func(*html.Node)
	find = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "body" {
			body = node
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			find(child)
		}
	}
	find(root)
	var fragment bytes.Buffer
	for child := body.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&fragment, child); err != nil {
			return Result{}, err
		}
	}
	output := fragment.String()
	notice := "Rendered with Goldmark and sanitized HTML. Code blocks, tables and README details keep their formatting."
	if hidden > 0 {
		notice += fmt.Sprintf(" %d image(s) are shown as placeholders; enable images and provide an asset base URL to load them.", hidden)
	}
	if values["images"] == "allow" && images > 0 {
		notice += " Enabled images can contact third-party hosts."
	}
	document := "<!doctype html>\n<html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>WeMe Markdown preview</title><style>" + documentCSS + "</style></head><body><article>" + output + "</article></body></html>"
	return Result{Text: document, HTML: output, Title: "Your document, rendered", Filename: "weme-preview.html", Mime: "text/html", Language: "html", Notice: notice}, nil
}
