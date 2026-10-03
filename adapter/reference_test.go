package adapter

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func referenceTree(t *testing.T, node *html.Node, depth int, out *strings.Builder) {
	t.Helper()
	prefix := "| " + strings.Repeat("  ", depth)
	switch node.Type {
	case html.ElementNode:
		name := node.Data
		if node.Namespace != "" {
			name = node.Namespace + " " + name
		}
		fmt.Fprintf(out, "%s<%s>\n", prefix, name)
		attrs := append([]html.Attribute(nil), node.Attr...)
		slices.SortFunc(attrs, func(a, b html.Attribute) int { return strings.Compare(a.Key, b.Key) })
		for _, attr := range attrs {
			name := attr.Key
			if attr.Namespace != "" {
				name = attr.Namespace + " " + name
			}
			fmt.Fprintf(out, "%s  %s=\"%s\"\n", prefix, name, attr.Val)
		}
	case html.TextNode:
		fmt.Fprintf(out, "%s\"%s\"\n", prefix, node.Data)
	default:
		t.Fatalf("unsupported reference node kind %v", node.Type)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		referenceTree(t, child, depth+1, out)
	}
}

func TestHTML5LibAdoptionCorpus(t *testing.T) {
	data, err := os.ReadFile("testdata/adoption01.dat")
	if err != nil {
		t.Fatal(err)
	}
	cases := strings.Split(string(data), "#data\n")[1:]
	for index, c := range cases {
		t.Run(fmt.Sprint(index+1), func(t *testing.T) {
			input, remaining, ok := strings.Cut(c, "\n#errors\n")
			if !ok {
				t.Fatal("missing error section")
			}
			metadata, expected, ok := strings.Cut(remaining, "#document\n")
			if !ok {
				t.Fatal("missing document section")
			}
			var root Node
			var err error
			if _, context, fragment := strings.Cut(metadata, "#document-fragment\n"); fragment {
				root, err = ParseFragment(input, strings.TrimSpace(context), 4096, 1000, 256)
			} else {
				root, err = Parse(input, 4096, 1000, 256)
			}
			if err != nil {
				t.Fatal(err)
			}
			var result strings.Builder
			for child := root.n.FirstChild; child != nil; child = child.NextSibling {
				referenceTree(t, child, 0, &result)
			}
			if strings.TrimRight(result.String(), "\n") != strings.TrimRight(expected, "\n") {
				t.Fatalf("input %q\ngot:\n%s\nwant:\n%s", input, result.String(), expected)
			}
		})
	}
}
