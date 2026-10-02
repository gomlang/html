# HTML utilities

Pure GoML, independently versioned HTML utilities. This module owns shared
escaping mechanics, not template context analysis or an HTML sanitizer.

`escape(value, max_bytes)` escapes ampersand, angle brackets and both quotes,
using numeric quote references. `escape_with(value, options, max_bytes)` selects
apostrophe escaping and numeric versus named double-quote references explicitly.
Negative limits fail; exact output length is checked before output allocation.
`escaped_len` computes that byte length with overflow checking.

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
digit runs. `finish()` applies the same EOF rules as whole-string decoding:
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

## Development and examples

Requires GoML 0.1.56 or newer. The `examples/basic/` example shares the root manifest. From the library root, run:

```sh
goml run --example basic
goml test
goml verify --timeout 300s
```

`goml test` builds the example and runs its tests. `goml verify` repeats the example checks as an independent module against an isolated registry snapshot. `(cd ../verification && just ecosystem-test html)` also retains the library-specific smoke and compatibility checks.
