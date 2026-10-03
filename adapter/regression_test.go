package adapter

import (
	"strings"
	"testing"
)

func TestRawTextInnerSerialization(t *testing.T) {
	cases := []struct{ input, selector, expected string }{
		{`<script>if (a < b && c) {}</script>`, "script", `if (a < b && c) {}`},
		{`<style>a>b { content: "&" }</style>`, "style", `a>b { content: "&" }`},
		{`<noscript>a < b &amp;</noscript>`, "noscript", `a < b &amp;`},
		{`<iframe>a < b &amp;</iframe>`, "iframe", `a < b &amp;`},
		{`<textarea>a &lt; b &amp;</textarea>`, "textarea", `a &lt; b &amp;`},
		{`<svg><script>a &lt; b &amp;</script></svg>`, "script", `a &lt; b &amp;`},
		{`<svg><style>a &lt; b &amp;</style></svg>`, "style", `a &lt; b &amp;`},
		{`<svg><foreignObject><script>a < b && c</script></foreignObject></svg>`, "script", `a < b && c`},
		{`<math><annotation-xml encoding="text/html"><script>a < b && c</script></annotation-xml></math>`, "script", `a < b && c`},
		{`<plaintext>a < b &amp;`, "plaintext", `a < b &amp;`},
		{`<svg><plaintext>a &lt; b &amp;</plaintext></svg>`, "plaintext", `a &lt; b &amp;`},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			root, err := ParseFragment(tc.input, "div", 4096, 100, 64)
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Select(root, tc.selector, 4096, 100, true)
			if err != nil || len(nodes) != 1 {
				t.Fatalf("selection: %d, %v", len(nodes), err)
			}
			for _, limit := range []int{4096, len(tc.expected)} {
				got, err := Render(nodes[0], true, limit)
				if err != nil || got != tc.expected {
					t.Fatalf("limit %d: %q, %v; want %q", limit, got, err, tc.expected)
				}
			}
			if result, err := Render(nodes[0], true, len(tc.expected)-1); result != "" || ErrorCode(err) != 5 {
				t.Fatalf("short limit: %q, %v", result, err)
			}
		})
	}
	for _, context := range []string{"script", "style", "noscript", "iframe"} {
		t.Run("fragment-"+context, func(t *testing.T) {
			input := `if (a < b && c) {}`
			root, err := ParseFragment(input, context, 4096, 100, 64)
			if err != nil {
				t.Fatal(err)
			}
			for _, inner := range []bool{false, true} {
				out, err := Render(root, inner, len(input))
				if err != nil || out != input {
					t.Fatalf("fragment serialization: %q, %v", out, err)
				}
			}
			parent := Parent(FirstChild(root))
			if out, err := Render(parent, false, len(input)); err != nil || out != input {
				t.Fatalf("retained context after navigation: %q, %v", out, err)
			}
		})
	}
}

func TestSelectorAdmissionRejectsExponentialQueries(t *testing.T) {
	root, err := ParseFragment(strings.Repeat("<div>", 40)+"leaf"+strings.Repeat("</div>", 40), "div", 1024, 100, 64)
	if err != nil {
		t.Fatal(err)
	}
	chain := "absent" + strings.Repeat(" div", 20)
	for _, query := range []string{
		chain,
		"absent" + strings.Repeat("/**/div", 20),
		"absent" + strings.Repeat("\tdiv", 20),
		":not(:not(" + chain + "))",
		":has(" + chain + ")",
		strings.Repeat(":has(", 12) + "absent" + strings.Repeat(")", 12),
	} {
		t.Run(query, func(t *testing.T) {
			for _, first := range []bool{false, true} {
				result, err := Select(root, query, 4096, 100, first)
				if len(result) != 0 || ErrorCode(err) != 11 {
					t.Fatalf("expected work rejection, got %d %v", len(result), err)
				}
			}
			leaf := root
			for !IsNil(FirstChild(leaf)) {
				leaf = FirstChild(leaf)
			}
			if result, err := Matches(Parent(leaf), query, 4096); result || ErrorCode(err) != 11 {
				t.Fatalf("single-node match must use document structural bounds: %v %v", result, err)
			}
		})
	}
	siblings, err := ParseFragment(strings.Repeat("<div>leaf</div>", 40), "div", 1024, 100, 64)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"absent" + strings.Repeat(" ~ div", 20), ":not(:not(absent" + strings.Repeat(" ~ div", 20) + "))"} {
		if result, err := Select(siblings, query, 4096, 100, false); len(result) != 0 || ErrorCode(err) != 11 {
			t.Fatalf("sibling query must be rejected: %d, %v", len(result), err)
		}
	}
	for _, query := range []string{
		`absent` + strings.Repeat(` \64 iv`, 20),
		`:\68 as(` + chain + ")",
		`:\6e ot(:\6e ot(` + chain + "))",
		`:matches(") ` + chain,
		`[title#="x"] ` + chain,
		`/*"*/` + strings.Repeat(":not(", 40) + "p" + strings.Repeat(")", 40),
	} {
		if result, err := Select(root, query, 4096, 100, false); len(result) != 0 || ErrorCode(err) != 6 {
			t.Fatalf("unsupported/bypassing syntax must be rejected: %q: %d, %v", query, len(result), err)
		}
	}
}

func TestSelectorAdmissionPreservesCommonQueries(t *testing.T) {
	root, err := ParseFragment(`<article lang="en"><h2><a href="/a">title</a></h2><ul><li class="x">a</li><li>b</li><li class="x">c</li></ul><p title='a"b'>text</p></article>`, "div", 1000, 100, 30)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		count int
	}{
		{"article > h2 a[href]", 1}, {"li:nth-child(2n + 1)", 2}, {"li:not(.x)", 1},
		{"article:has(h2 > a)", 1}, {"li:first-child ~ li", 2}, {"li + li", 2},
		{"article/**/>/**/h2 a", 1}, {"p:lang(en)", 1}, {`[title="a\"b"]`, 1},
		{"h2, li.x", 3}, {"li:only-child", 0},
	} {
		result, err := Select(root, tc.query, 4096, 100, false)
		if err != nil || len(result) != tc.count {
			t.Fatalf("%s: %d, %v", tc.query, len(result), err)
		}
	}
}
