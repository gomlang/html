package adapter

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/andybalholm/cascadia"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Node struct {
	n *html.Node
}

type Attribute struct {
	a html.Attribute
}

type failure struct {
	code int
	text string
}

func (e failure) Error() string { return e.text }

func ErrorCode(err error) int {
	if err == nil {
		return 0
	}
	var e failure
	if errors.As(err, &e) {
		return e.code
	}
	return 7
}

func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func checkInput(input string, maxInput, maxNodes, maxDepth int) error {
	if maxInput < 0 || maxNodes < 1 || maxDepth < 0 || maxDepth > 512 {
		return failure{1, "invalid DOM limits: input >= 0, nodes >= 1, depth in 0..512 required"}
	}
	if len(input) > maxInput {
		return failure{2, "HTML input byte limit exceeded"}
	}
	if !utf8.ValidString(input) {
		return failure{7, "HTML input is not UTF-8"}
	}
	return nil
}

func checkTree(root *html.Node, maxNodes, maxDepth int) error {
	count, depth := 0, 0
	for n := root; n != nil; {
		count++
		if count > maxNodes {
			return failure{3, "HTML node limit exceeded"}
		}
		if depth > maxDepth {
			return failure{4, "HTML tree depth limit exceeded"}
		}
		if n.FirstChild != nil {
			n = n.FirstChild
			depth++
			continue
		}
		for n != root && n.NextSibling == nil {
			n = n.Parent
			depth--
		}
		if n == root {
			break
		}
		n = n.NextSibling
	}
	return nil
}

func Parse(input string, maxInput, maxNodes, maxDepth int) (Node, error) {
	if err := checkInput(input, maxInput, maxNodes, maxDepth); err != nil {
		return Node{}, err
	}
	n, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return Node{}, failure{7, err.Error()}
	}
	if err := checkTree(n, maxNodes, maxDepth); err != nil {
		return Node{}, err
	}
	return Node{n}, nil
}

func ParseFragment(input, context string, maxInput, maxNodes, maxDepth int) (Node, error) {
	if err := checkInput(input, maxInput, maxNodes, maxDepth); err != nil {
		return Node{}, err
	}
	a := atom.Lookup([]byte(context))
	if !validContext(context) {
		return Node{}, failure{8, "fragment context must be a recognized lowercase HTML element"}
	}
	nodes, err := html.ParseFragment(strings.NewReader(input), &html.Node{
		Type: html.ElementNode, Data: context, DataAtom: a,
	})
	if err != nil {
		return Node{}, failure{7, err.Error()}
	}
	root := &html.Node{Type: html.DocumentNode}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	if err := checkTree(root, maxNodes, maxDepth); err != nil {
		return Node{}, err
	}
	return Node{root}, nil
}

func validContext(context string) bool {
	switch context {
	case "a", "abbr", "address", "area", "article", "aside", "audio", "b", "base", "bdi", "bdo", "blockquote", "body", "br", "button", "canvas", "caption", "cite", "code", "col", "colgroup", "data", "datalist", "dd", "del", "details", "dfn", "dialog", "div", "dl", "dt", "em", "embed", "fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hgroup", "hr", "html", "i", "iframe", "img", "input", "ins", "kbd", "label", "legend", "li", "link", "main", "map", "mark", "menu", "meta", "meter", "nav", "noscript", "object", "ol", "optgroup", "option", "output", "p", "picture", "pre", "progress", "q", "rp", "rt", "ruby", "s", "samp", "script", "search", "section", "select", "slot", "small", "source", "span", "strong", "style", "sub", "summary", "sup", "table", "tbody", "td", "template", "textarea", "tfoot", "th", "thead", "time", "title", "tr", "track", "u", "ul", "var", "video", "wbr":
		return true
	default:
		return false
	}
}

func IsNil(node Node) bool { return node.n == nil }

func Kind(node Node) int {
	if node.n == nil {
		return 0
	}
	return int(node.n.Type)
}

func Data(node Node) string {
	if node.n == nil {
		return ""
	}
	return node.n.Data
}

func Namespace(node Node) string {
	if node.n == nil {
		return ""
	}
	return node.n.Namespace
}

func Parent(node Node) Node {
	if node.n == nil {
		return Node{}
	}
	return Node{node.n.Parent}
}

func FirstChild(node Node) Node {
	if node.n == nil {
		return Node{}
	}
	return Node{node.n.FirstChild}
}

func NextSibling(node Node) Node {
	if node.n == nil {
		return Node{}
	}
	return Node{node.n.NextSibling}
}

func SameNode(left, right Node) bool { return left.n == right.n }

func Attributes(node Node) []Attribute {
	if node.n == nil {
		return nil
	}
	result := make([]Attribute, len(node.n.Attr))
	for i, a := range node.n.Attr {
		result[i] = Attribute{a}
	}
	return result
}

func AttributeName(a Attribute) string      { return a.a.Key }
func AttributeValue(a Attribute) string     { return a.a.Val }
func AttributeNamespace(a Attribute) string { return a.a.Namespace }

func AttributeValueByName(node Node, namespace, key string) (string, bool) {
	if node.n != nil {
		for _, a := range node.n.Attr {
			if a.Namespace == namespace && a.Key == key {
				return a.Val, true
			}
		}
	}
	return "", false
}

type limitedWriter struct {
	output strings.Builder
	limit  int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.output.Len() {
		return 0, failure{5, "HTML output byte limit exceeded"}
	}
	return w.output.Write(p)
}

func (w *limitedWriter) WriteString(s string) (int, error) {
	if len(s) > w.limit-w.output.Len() {
		return 0, failure{5, "HTML output byte limit exceeded"}
	}
	return w.output.WriteString(s)
}

func Render(node Node, inner bool, maxOutput int) (string, error) {
	if maxOutput < 0 {
		return "", failure{1, "negative output byte limit"}
	}
	w := &limitedWriter{limit: maxOutput}
	if node.n == nil {
		return "", nil
	}
	if inner {
		for c := node.n.FirstChild; c != nil; c = c.NextSibling {
			if err := html.Render(w, c); err != nil {
				return "", err
			}
		}
	} else if err := html.Render(w, node.n); err != nil {
		return "", err
	}
	return w.output.String(), nil
}

func Text(node Node, maxOutput int) (string, error) {
	if maxOutput < 0 {
		return "", failure{1, "negative output byte limit"}
	}
	w := &limitedWriter{limit: maxOutput}
	if node.n == nil {
		return "", nil
	}
	for n := node.n; n != nil; {
		if n.Type == html.TextNode {
			if _, err := w.WriteString(n.Data); err != nil {
				return "", err
			}
		}
		if n.FirstChild != nil {
			n = n.FirstChild
			continue
		}
		for n != node.n && n.NextSibling == nil {
			n = n.Parent
		}
		if n == node.n {
			break
		}
		n = n.NextSibling
	}
	return w.output.String(), nil
}

func selector(query string, maxBytes int) (cascadia.SelectorGroup, error) {
	if maxBytes < 0 {
		return nil, failure{1, "negative selector byte limit"}
	}
	if len(query) > maxBytes || len(query) > 4096 {
		return nil, failure{6, "CSS selector exceeds byte limit (hard maximum 4096)"}
	}
	depth := 0
	var quote rune
	escaped := false
	for _, r := range query {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
		} else if r == '(' || r == '[' {
			depth++
			if depth > 32 {
				return nil, failure{6, "CSS selector nesting exceeds 32"}
			}
		} else if r == ')' || r == ']' {
			depth--
		}
	}
	s, err := cascadia.ParseGroup(query)
	if err != nil {
		return nil, failure{6, err.Error()}
	}
	return s, nil
}

func Matches(node Node, query string, maxBytes int) (bool, error) {
	s, err := selector(query, maxBytes)
	if err != nil {
		return false, err
	}
	return node.n != nil && s.Match(node.n), nil
}

func Select(node Node, query string, maxBytes, maxMatches int, first bool) ([]Node, error) {
	if maxMatches < 0 {
		return nil, failure{1, "negative selector match limit"}
	}
	s, err := selector(query, maxBytes)
	if err != nil {
		return nil, err
	}
	var result []Node
	if node.n == nil {
		return result, nil
	}
	for n := range node.n.Descendants() {
		if s.Match(n) {
			if len(result) == maxMatches {
				return nil, failure{9, "CSS selector match limit exceeded"}
			}
			result = append(result, Node{n})
			if first {
				break
			}
		}
	}
	return result, nil
}

func makePolicy(rich bool) *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.SkipElementsContent("script", "style", "iframe", "object", "embed", "svg", "math", "template", "noscript", "noembed", "noframes", "xmp", "plaintext")
	if rich {
		p.AllowElements("p", "br", "hr", "b", "strong", "i", "em", "s", "del", "u", "sub", "sup", "blockquote", "pre", "code", "ul", "ol", "li", "dl", "dt", "dd", "h1", "h2", "h3", "h4", "h5", "h6", "table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption", "a", "span", "div")
		p.AllowAttrs("href").OnElements("a")
		p.AllowAttrs("title").Globally()
		p.RequireParseableURLs(true)
		p.AllowRelativeURLs(true)
		p.AllowURLSchemes("http", "https", "mailto")
		p.RequireNoFollowOnLinks(true)
		p.RequireNoReferrerOnLinks(true)
	}
	return p
}

var textPolicy = makePolicy(false)
var richPolicy = makePolicy(true)

func Sanitize(input string, rich bool, maxInput, maxNodes, maxDepth, maxOutput int) (string, error) {
	if maxOutput < 0 {
		return "", failure{1, "negative output byte limit"}
	}
	n, err := ParseFragment(input, "div", maxInput, maxNodes, maxDepth)
	if err != nil {
		return "", err
	}
	normalized, err := Render(n, true, maxOutput)
	if err != nil {
		return "", err
	}
	p := textPolicy
	if rich {
		p = richPolicy
	}
	for range 4 {
		w := &limitedWriter{limit: maxOutput}
		if err := p.SanitizeReaderToWriter(strings.NewReader(normalized), w); err != nil {
			return "", err
		}
		clean := w.output.String()
		parsed, err := ParseFragment(clean, "div", maxOutput, maxNodes, maxDepth)
		if err != nil {
			return "", err
		}
		canonical, err := Render(parsed, true, maxOutput)
		if err != nil {
			return "", err
		}
		if canonical == normalized && canonical == clean {
			return canonical, nil
		}
		normalized = canonical
	}
	return "", failure{10, "HTML sanitization did not stabilize after four passes"}
}
