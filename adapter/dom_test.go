package adapter

import (
	"strings"
	"sync"
	"testing"

	"golang.org/x/net/html"
)

func TestHTMLTreeConstruction(t *testing.T) {
	cases := []struct{ input, context, output string }{
		{"<p>one<div>two</div>three", "div", "<p>one</p><div>two</div>three"},
		{"<b><i>one</b>two</i>", "div", "<b><i>one</i></b><i>two</i>"},
		{"<table>before<tr><td>cell</td></tr>after</table>", "div", "beforeafter<table><tbody><tr><td>cell</td></tr></tbody></table>"},
		{"<tr><td>A<td>B", "table", "<tbody><tr><td>A</td><td>B</td></tr></tbody>"},
		{"<option>A<option>B", "select", "<option>A</option><option>B</option>"},
		{"&lt;b&gt;X&lt;/b&gt;", "textarea", "&lt;b&gt;X&lt;/b&gt;"},
		{"<svg viewBox='0 0 1 1'><circle/></svg>", "div", `<svg viewBox="0 0 1 1"><circle></circle></svg>`},
		{"<p title='&quot;&amp;'>界&#x1f642;", "div", `<p title="&#34;&amp;">界🙂</p>`},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n, err := ParseFragment(tc.input, tc.context, 4096, 100, 64)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Render(n, true, 4096)
			if err != nil || got != tc.output {
				t.Fatalf("render = %q, %v; want %q", got, err, tc.output)
			}
		})
	}
	doc, err := Parse("<!doctype html><title>T</title><p>A", 100, 20, 10)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Render(doc, false, 200)
	if err != nil || got != "<!DOCTYPE html><html><head><title>T</title></head><body><p>A</p></body></html>" {
		t.Fatalf("document = %q, %v", got, err)
	}
}

func TestSelectorsNavigationAndBounds(t *testing.T) {
	n, err := ParseFragment(`<ul id="list"><li class="x" data-n="one">A</li><li>B<b>!</b></li><li class="x">C</li></ul>`, "div", 1000, 100, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		count int
	}{
		{"#list > li.x", 2}, {"li:nth-child(2)", 1}, {"li + li", 2}, {"li:first-child ~ li", 2},
		{`[data-n^="on"]`, 1}, {"li:not(.x)", 1}, {"li, .x", 3}, {"ul li b", 1},
	} {
		result, err := Select(n, tc.query, 4096, 100, false)
		if err != nil || len(result) != tc.count {
			t.Fatalf("%s: %d, %v", tc.query, len(result), err)
		}
	}
	items, _ := Select(n, "li", 4096, 100, false)
	if Data(Parent(items[0])) != "ul" || !SameNode(NextSibling(items[0]), items[1]) {
		t.Fatal("navigation identity")
	}
	if value, ok := AttributeValueByName(items[0], "", "data-n"); !ok || value != "one" {
		t.Fatal("attribute lookup")
	}
	if text, err := Text(items[1], 2); err != nil || text != "B!" {
		t.Fatalf("text %q, %v", text, err)
	}
	if _, err := Text(items[1], 1); ErrorCode(err) != 5 {
		t.Fatal("text limit")
	}
	if result, err := Select(n, "li", 10, 1, false); len(result) != 0 || ErrorCode(err) != 9 {
		t.Fatal("selection must not return a partial result")
	}
	if result, err := Select(n, "li", 10, 1, true); len(result) != 1 || err != nil {
		t.Fatal("first selection must stop after one")
	}
	for _, q := range []string{"[", "a::unknown", strings.Repeat(":not(", 40) + "p" + strings.Repeat(")", 40), strings.Repeat("a", 4097)} {
		if _, err := Select(n, q, 5000, 10, false); ErrorCode(err) != 6 {
			t.Fatalf("invalid selector accepted: %q %v", q, err)
		}
	}
	if found, _ := Select(items[1], "li", 10, 10, false); len(found) != 0 {
		t.Fatal("select must exclude scope node")
	}
	if match, _ := Matches(items[1], "li", 10); !match {
		t.Fatal("matches must include scope node")
	}
	if _, err := Parse("abc", 2, 100, 10); ErrorCode(err) != 2 {
		t.Fatal("input limit")
	}
	if _, err := Parse("", 0, 3, 10); ErrorCode(err) != 3 {
		t.Fatal("implicit nodes counted")
	}
	if _, err := Parse("", 0, 4, 1); ErrorCode(err) != 4 {
		t.Fatal("implicit depth counted")
	}
	for _, context := range []string{"not-a-real-tag", "href", "onclick", "svg", "DIV"} {
		if _, err := ParseFragment("", context, 0, 1, 0); ErrorCode(err) != 8 {
			t.Fatalf("invalid context %q", context)
		}
	}
	if _, err := Parse(string([]byte{0xff}), 2, 100, 10); ErrorCode(err) != 7 {
		t.Fatal("invalid UTF-8")
	}
	if _, err := Parse("", 0, 100, 513); ErrorCode(err) != 1 {
		t.Fatal("depth ceiling")
	}
}

func assertSafeTree(t *testing.T, output string) {
	t.Helper()
	root, err := html.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	for node := range root.Descendants() {
		if node.Namespace != "" {
			t.Fatalf("foreign namespace in %q", output)
		}
		if node.Type != html.ElementNode {
			continue
		}
		switch node.Data {
		case "script", "style", "svg", "math", "iframe", "object", "embed", "form", "input", "img", "template", "noscript":
			t.Fatalf("unsafe element %s in %q", node.Data, output)
		}
		for _, attr := range node.Attr {
			if attr.Key != "href" && attr.Key != "title" && attr.Key != "rel" {
				t.Fatalf("unexpected attribute %s in %q", attr.Key, output)
			}
			if attr.Key == "href" {
				lower := strings.ToLower(attr.Val)
				for _, bad := range []string{"javascript:", "vbscript:", "data:"} {
					if strings.Contains(lower, bad) {
						t.Fatalf("unsafe URL in %q", output)
					}
				}
			}
		}
	}
}

func TestSanitizerHostileVectors(t *testing.T) {
	vectors := []string{
		`<script>alert(1)</script><p onclick="alert(1)">ok</p>`,
		`<a href="javascript:alert(1)">link</a>`,
		`<a href="jav&#x61;script:alert(1)">link</a>`,
		"<a href=\"java\tscript:alert(1)\">link</a>",
		"<a href=\"java\nscript:alert(1)\">link</a>",
		`<a href="&#14;javascript:alert(1)">link</a>`,
		`<a href="data:text/html,&lt;script&gt;alert(1)&lt;/script&gt;">link</a>`,
		`<img src=x onerror=alert(1)><svg><a xlink:href="javascript:alert(1)">x</a></svg>`,
		`<math><mtext><table><mglyph><style><!--</style><img title="--><img src=1 onerror=alert(1)>">`,
		`<svg></p><style><a id="</style><img src=1 onerror=alert(1)>">`,
		`<noscript><p title="</noscript><img src=x onerror=alert(1)>">x</p>`,
		`<svg><foreignObject><p onmouseover=alert(1)>x</p></foreignObject></svg>`,
		`<math><annotation-xml encoding="text/html"><script>alert(1)</script></annotation-xml></math>`,
		`<template><p><img src=x onerror=alert(1)></p></template>safe`,
		`<p style="background:url(javascript:alert(1))" id="location" name="cookie" data-x="javascript:alert(1)">text</p>`,
		`<form action="javascript:alert(1)"><input formaction="javascript:alert(1)"></form>`,
		`<a href="https://example.com" href="javascript:alert(1)">ok</a>`,
		`<a href="javascript:alert(1)" href="https://example.com">ok</a>`,
		`<table><tr><td><svg><script>alert(1)</script></svg>cell</td></tr></table>`,
		`<plaintext><script>alert(1)</script>`,
	}
	for _, input := range vectors {
		t.Run(input, func(t *testing.T) {
			clean, err := Sanitize(input, true, 8192, 1000, 100, 32768)
			if err != nil {
				t.Fatal(err)
			}
			assertSafeTree(t, clean)
			twice, err := Sanitize(clean, true, 32768, 1000, 100, 32768)
			if err != nil || clean != twice {
				t.Fatalf("not idempotent: %q -> %q, %v", clean, twice, err)
			}
		})
	}
	for _, tc := range []struct {
		input, output string
		rich          bool
	}{
		{`<p onclick="x">hello <strong>world</strong></p>`, `<p>hello <strong>world</strong></p>`, true},
		{`<p>Hello &amp; <b>world</b></p><script>secret</script>`, `Hello &amp; world`, false},
		{`<table><tr><td>A</td></tr></table>`, `<table><tbody><tr><td>A</td></tr></tbody></table>`, true},
		{`<a href="https://example.com">x</a>`, `<a href="https://example.com" rel="nofollow noreferrer">x</a>`, true},
	} {
		clean, err := Sanitize(tc.input, tc.rich, 4096, 100, 32, 8192)
		if err != nil || clean != tc.output {
			t.Fatalf("%q => %q, %v; want %q", tc.input, clean, err, tc.output)
		}
	}
	if result, err := Sanitize("abc", true, 3, 2, 1, 2); result != "" || ErrorCode(err) != 5 {
		t.Fatal("sanitizer must not return partial output")
	}
}

func TestConcurrentImmutableOperations(t *testing.T) {
	n, err := ParseFragment(`<p class="x">hello <b>world</b></p>`, "div", 1000, 100, 30)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				if matches, err := Select(n, "p.x > b", 100, 10, false); err != nil || len(matches) != 1 {
					t.Errorf("select %v", err)
				}
				if text, err := Text(n, 100); err != nil || text != "hello world" {
					t.Errorf("text %q %v", text, err)
				}
				if _, err := Sanitize(`<p onclick="x">ok</p>`, true, 100, 100, 30, 100); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
}
