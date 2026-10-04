# HTML utilities

Independently versioned HTML utilities. The root `ecosystem::html` package keeps
its pure GoML escaping and entity APIs. `ecosystem::html::dom` adds HTML5 document
parsing, CSS selectors and conservative allowlist sanitization through pinned Go
backends. Contextual template analysis remains the caller's responsibility.

The module now declares a native Go adapter. With the current GoML driver, all
consuming modules need a module-root `go.mod`, including applications that only
use entity helpers or depend on HTML through Template/Markdown. See the native
setup below when upgrading.

`escape(value, max_bytes)` escapes ampersand, angle brackets and both quotes,
using numeric quote references. `escape_with(value, options, max_bytes)` selects
apostrophe escaping and numeric versus named double-quote references explicitly.
Negative limits fail; exact output length is checked before output allocation.
`escaped_len` computes that byte length with overflow checking.
These bounded helpers reuse the original string when no character needs
escaping, without allocating an output buffer.

`escape_text` is the unbounded legacy-compatible text helper: double quotes become
`&quot;`, apostrophes remain unchanged. It is not suitable for single-quoted
attribute interpolation. All helpers preserve other UTF-8 bytes, including NUL,
and escape ampersands in already-escaped text again. Output is not marked safe.
Unquoted attributes, JavaScript, CSS, URL schemes and contextual template safety
need their own policy; escaping alone does not make those contexts safe.

Template and Markdown retain compatible public wrappers backed by this module.
`named_entity(name)` looks up one of the 2,125 canonical HTML5 names, without
an ampersand or trailing semicolon, and returns its one- or two-scalar replacement.
Lookup is case-sensitive and unknown names return None. Markdown's stricter
entity parser uses this same table. This lookup does not parse character references
or apply HTML's semicolon-optional or attribute-context rules by itself.

`unescape(value, max_bytes)` decodes character references in HTML text.
`unescape_with(value, DecodeContext::Attribute, max_bytes)` selects attribute-value
rules: a semicolon-free named reference followed by an ASCII letter, digit or `=`
is not consumed. Both use longest matching names, including the 106 legacy names
that permit missing semicolons. Unknown names and malformed numeric syntax remain
literal. Decoding is single-pass: `&amp;lt;` becomes `&lt;`, not `<`.

Numeric references accept decimal or hexadecimal digits with optional semicolons.
Zero, surrogates and out-of-range values become U+FFFD; C1 values use HTML's
Windows-1252 replacement mapping. Arbitrarily long digit sequences saturate without
integer wraparound. Other control/noncharacter scalars are retained. There are
intentional differences from Go 1.26's numeric-reference parser: a single decimal
digit without a semicolon is decoded both at EOF and before a non-digit, so `&#1!`
becomes U+0001 followed by `!`, and `&#0&amp;` becomes U+FFFD followed by `&`.
Go incorrectly leaves those numeric references literal because its consumed-length
check mistakes them for missing digits. Conversely, `&#x;` and `&#X;` contain no
hexadecimal digits and remain literal here; Go incorrectly converts them to U+FFFD.
Overflowing digit sequences also do not reproduce Go's 32-bit integer wraparound.

`decoded_len(value, context)` computes the exact decoded UTF-8 byte length without
allocating the decoded output, with checked arithmetic. Both decoders preflight
the same reference rules and reject an oversized result before constructing its
output. Literal runs are copied together; reference-free input is returned directly.
This uses two scans for accepted reference-containing input.

Negative limits fail and errors return no partial string. Limits bound logical output bytes, not input length or total
process memory. Decoding does not sanitize markup, check URL schemes or validate
which tokenizer state a caller is in. These utilities do not tokenize HTML.

`Decoder::new(context, DecodeLimits)` provides incremental character-reference
conversion. `push(chunk)` accepts complete UTF-8 strings and returns available
output; references may cross any string boundary. It retains at most 32 pending
name/prefix bytes and a saturated numeric accumulator, even for arbitrarily long
digit runs. With no reference pending, chunks containing no ampersand are returned
directly without allocating a decoded copy. `finish()` applies the same EOF rules
as whole-string decoding:
semicolon-free legacy/numeric references decode, while `&`, `&#` and `&#x` remain
literal. A successful finish closes the decoder; later push/finish calls return
`DecodeError::Finished`.

`DecodeLimits::standard()` allows 16 MiB cumulative input and 16 MiB cumulative
output, independently configurable to nonnegative byte counts. A call first checks
its complete input size, then checks every output append. A failed call returns no
partial chunk, commits no counters/state, and permanently poisons the decoder;
all subsequent calls return that first error. Previously returned chunks remain
valid. `input_bytes`, `output_bytes` and `buffered_bytes` expose accepted input,
returned output and retained prefix bytes. Copies share mutable state; serialize
access and construct a new decoder for an independent stream. This is character
reference streaming, not HTML tokenization; callers still select text/attribute
context and provide valid UTF-8 chunks.

Boundary expectations follow the HTML Standard's
[numeric reference states](https://html.spec.whatwg.org/multipage/parsing.html#numeric-character-reference-state)
and [recovery rules](https://html.spec.whatwg.org/multipage/parsing.html#numeric-character-reference-end-state):
missing semicolons do not prevent numeric decoding, absent hexadecimal digits
leave literal text, and control/noncharacter parse errors do not imply deleting
the scalar. The utility returns decoded text rather than tokenizer diagnostics.

The independently sourced data lives in [data/entities.json](data/entities.json),
with provenance in [data/README.md](data/README.md) and its retained
[license](LICENSE.entities.txt). The existing native generator under
`../markdown/tools` generates both this module's lookup and Markdown's compatibility
wrapper; no Python installation or runtime data file access is required by HTML.

Run `(cd ../verification && just ecosystem-test html template markdown)` from this library repository.

## DOM, CSS selectors and sanitization

```goml
use ecosystem::html::dom;

fn extract(input: string) -> Result[Vec[string], dom::Error] {
    let document = dom::parse(input, dom::Limits::standard())?;
    let titles = Vec::new();
    for heading in document.select("article > h2")? {
        titles.push(heading.text()?);
    }
    Ok(titles)
}
```

`parse(input, limits)` uses the HTML5 tree constructor with scripting enabled:
it supplies implicit `html`, `head` and `body` nodes, closes paragraphs, repairs
misnested formatting and applies table foster parenting. HTML syntax recovery is
part of parsing; malformed HTML ordinarily produces a recovered tree rather than
a validation error. This does not execute scripts or load referenced resources.

`parse_fragment(input, context, limits)` returns a synthetic document whose
children are the fragment nodes. Serialization retains the original context, so
a `script` fragment preserves literal `<` and `&` in its text. The context is a recognized lowercase HTML
element, for example `div`, `table`, `select` or `textarea`; these contexts have
different parsing rules. Foreign namespace and custom-element contexts are not
accepted. Parsed documents may contain SVG/MathML nodes, and `namespace()` plus
`attribute_ns(namespace, name)` preserve their namespace information.

`Document::root`, `select`, `select_first` and `to_html` expose the immutable tree.
Nodes provide `kind`, `name`, `namespace`, `value`, `parent`, `first_child`,
`next_sibling`, `children`, `same_node`, `attributes`, `attribute` and
`attribute_ns`. Attributes are copied records with `namespace`, `name`, `value`.
HTML element/attribute names are normalized by the parser; attribute lookup is
case-sensitive and returns the first matching attribute. `value()` reads text or
comment data; `text()` concatenates descendant text without layout whitespace or
visibility processing. It includes script/style text when present in an
unsanitized tree. `inner_html` and `outer_html` produce HTML serialization rather
than preserving original formatting. Inner serialization preserves the parent's
raw-text and namespace rules: HTML script/style text stays literal, while SVG
script/style text remains escaped. These methods do not sanitize their output.

`select` returns matching descendants in document order, excluding the node on
which it is called; selector groups deduplicate matches. `select_first` stops at
the first match. `matches` tests the node itself. The supported bounded selector
subset uses
[Cascadia](https://github.com/andybalholm/cascadia), including element/ID/class,
attribute operators, combinators, groups, `:not`, `:has`, `:haschild`, nth/first/
last/only child/type pseudo-classes, `:empty`, `:root`, `:lang`, `:link`, `:input`,
`:checked`, `:enabled` and `:disabled`. Escapes are supported inside quoted
attribute values only. Regex attribute matching and Cascadia's text/regex pseudo-
classes are excluded. Invalid or unsupported syntax returns `ErrorKind::Selector`.
This is static tree matching, without browser
layout, interactive pseudo-class state or a promise of complete Selectors Level 4
support. A selector can inspect ancestors outside the selected subtree, as with
ordinary descendant queries. Cascadia compares attribute local names without
namespace filtering: `[href]` can match SVG `xlink:href`, while `attribute("href")`
and `attribute_ns("xlink", "href")` remain distinct. Use the namespace-aware
accessors for decisions that depend on attribute namespaces.

`Limits::standard()` supplies:

| Limit | Default | Enforcement |
| --- | --- | --- |
| `max_input_bytes` | 1 MiB | Before parsing |
| `max_nodes` | 100,000 | After tree construction, including implicit/root nodes |
| `max_depth` | 256 | Root depth is zero; checked after construction; maximum setting 512 |
| `max_output_bytes` | 4 MiB | Bounded text/render/sanitizer writers |
| `max_selector_bytes` | 4,096 | Before selector compilation; absolute ceiling 4,096 |
| `max_matches` | 10,000 | Before appending each result; exceeding it returns no partial vector |

Nodes must be at least one; other limits must be nonnegative. Selector nesting
has an additional hard ceiling of 32. Before invoking the matcher, an admission
check estimates structural work using the immutable document's actual node count,
maximum depth and maximum child count. It starts with selector length times
document nodes (one starting node for `matches`), multiplies by ancestor/sibling
scan bounds for combinators and by traversal bounds for recursive/structural
pseudo-classes, including those nested in `:not` and `:has`. An estimate above
10,000,000 returns `SelectorWorkLimit` without matching or partial results. The
same guard applies to `select_first`. The estimate deliberately combines maxima
across the complete document and may reject a query whose actual evaluation would
be cheaper. This bounds admitted structural matching work; it is not a wall-clock
or instruction-count guarantee.

The HTML backend independently rejects an
open-element stack deeper than 512. Input limits bound admitted source bytes;
node/depth limits validate the resulting tree and do not cap parser allocations
or process memory before construction. There is no CPU-time deadline. Node
handles keep their document alive, share an immutable tree and support concurrent
read operations. They need no explicit close.

`sanitize(input, SanitizePolicy::RichText, limits)` returns HTML for an ordinary
HTML body/div context. It uses the HTML5 parser, [bluemonday](https://github.com/microcosm-cc/bluemonday),
then reparses and rechecks until both the allowlist and serialization stabilize.
It returns `UnstableSanitization` after four unsuccessful passes. Input, tree and
output budgets apply to intermediate normalization as well, so a large input
whose final cleaned result would be small may still fail a budget. Failures
return no partial string.

The rich-text policy allows paragraphs, headings, emphasis, lists, code,
blockquotes, tables, links, spans and divs. It permits `title` attributes and link
`href` values using HTTP, HTTPS, mailto or relative URLs, adding `nofollow
noreferrer` to retained links. It drops event handlers, CSS, IDs/names, data
attributes, forms, images and embedded media. Script, style, foreign SVG/MathML,
template, iframe and other active/rawtext containers lose their content. Protocol
relative links count as relative URLs; permitted destinations can still lead to
untrusted sites. This API is an HTML allowlist, not a link destination/tracking
policy.

`SanitizePolicy::TextOnly` removes markup and active-container content but returns
escaped HTML text: `A &amp; B` stays escaped for HTML insertion. To obtain literal
text, parse the result and call `root().text()`. Neither policy produces content
for JavaScript, CSS, attributes, URLs, foreign XML, table/select-specific insertion
contexts or a client framework's expression language. Do not interpolate the
result into those contexts or add untrusted markup after sanitizing. Sanitizer
regressions cover obfuscated URLs, rawtext, foreign content and mutation patterns;
they do not establish a guarantee for every browser or future browser behavior.

## Native setup and verification

DOM support uses `golang.org/x/net` v0.59.0, Cascadia v1.3.5 and bluemonday v1.0.27.
Dependencies and checksums are pinned in `go.mod`/`go.sum`; no cgo is needed. Go
1.26 or newer is required. Prepare the native dependencies with `go mod download`
from this repository before compilation in an environment without cached modules.

A consumer supplies its own minimal Go module, for example:

```go
module example.com/my-html-app

go 1.26.0
```

GoML resolves the native adapter from the selected GoML dependency and creates
managed requirements/replacements under the consumer's artifact directory. Do
not add machine-local `replace` paths to published manifests. Existing root
entity API signatures and behavior are retained; the additional Go module setup
is required even when the consumer does not import `dom`.

`goml bind-go bindings.json` regenerates the explicitly allowlisted raw boundary.
The handwritten adapter is in `adapter/`; `native/generated.go`,
`dom/bindings/generated.goml` and the ownership manifest are generated together.
Do not edit them directly. The public API performs explicit Go/GoML conversions.

```sh
go mod download
goml bind-go bindings.json
goml fmt --check
goml test
goml verify --timeout 300s
go test -race ./adapter
```

The `examples/dom/` consumer demonstrates extraction and sanitization. Native
tests include an independent 18-case html5lib adoption-agency fixture with its
license/provenance in `adapter/testdata`, fixed tree/render/selector expectations,
hostile sanitizer vectors and concurrent immutable operations. Existing entity
tests and the entity-only `examples/basic/` consumer remain in the test suite.

## Development and examples

Requires GoML 0.1.56 or newer. The `examples/basic/` example shares the root manifest. From the library root, run:

```sh
goml run --example basic
goml test
goml verify --timeout 300s
```

`goml test` builds the example and runs its tests. `goml verify` repeats the example checks as an independent module against an isolated registry snapshot. `(cd ../verification && just ecosystem-test html)` also retains the library-specific smoke and compatibility checks.
