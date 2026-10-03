# HTML DOM reference fixtures

`adoption01.dat` is the unmodified 18-case adoption-agency tree-construction
fixture from [html5lib-tests](https://github.com/html5lib/html5lib-tests/blob/master/tree-construction/adoption01.dat),
as vendored by `golang.org/x/net` v0.59.0. Its independently maintained expected
trees exercise misnested formatting, reconstruction, foster parenting and an
HTML fragment context. `reference_test.go` compares those trees directly to the
adapter's results; it does not generate expectations with the backend. The
upstream MIT license is retained in `LICENSE.html5lib`.

Fixture SHA-256: `b2aba05bd1d832f73a0c6103b3c8b151b283bab7c56887274d81f3a062c4963e`.

The additional fixed rendering cases in `dom_test.go` cover implicit paragraphs,
table/select/textarea contexts, entities and foreign attributes. The sanitizer
tests include obfuscated URL schemes, duplicate attributes, rawtext and
foreign-content/mutation-XSS patterns, including patterns documented by
[DOMPurify's regression fixtures](https://github.com/cure53/DOMPurify/blob/main/test/fixtures/expect.mjs).
They assert the result after HTML5 reparsing and repeated sanitization. This is
a targeted regression set, not the complete html5lib or DOMPurify conformance
suite, and it does not run a browser engine.
