# HTML DOM reference fixtures

`adoption01.dat` is the unmodified 18-case adoption-agency tree-construction
fixture from [html5lib-tests](https://github.com/html5lib/html5lib-tests/blob/master/tree-construction/adoption01.dat),
as vendored by `golang.org/x/net` v0.59.0. Its independently maintained expected
trees exercise misnested formatting, reconstruction, foster parenting and an
HTML fragment context. `../reference_test.goml` compares those trees directly to the
pure GoML parser's results; it does not generate expectations with the backend. The
upstream MIT license is retained in `LICENSE.html5lib`.

Fixture SHA-256: `b2aba05bd1d832f73a0c6103b3c8b151b283bab7c56887274d81f3a062c4963e`.

The additional fixed rendering cases in `../tree_regression_test.goml` cover implicit paragraphs,
table/select/textarea contexts, entities and foreign attributes. The sanitizer
tests include obfuscated URL schemes, duplicate attributes, rawtext and
foreign-content/mutation-XSS patterns, including patterns documented by
[DOMPurify's regression fixtures](https://github.com/cure53/DOMPurify/blob/main/test/fixtures/expect.mjs).
They assert the result after HTML5 reparsing and repeated sanitization. This is
a targeted regression set, not the complete html5lib or DOMPurify conformance
suite, and it does not run a browser engine.

The reference test embeds the 18 input/context/expected-tree triples verbatim from
this fixture, so tests do not depend on a runtime working directory or data-file
lookup. Regenerate those constants from this retained fixture when changing its
upstream version; never derive expected trees from the implementation under test.

`../sanitize_regression_test.goml` retains fixed outputs from the former adapter with x/net/html v0.59.0 and bluemonday v1.0.27, generated using Go 1.26. It covers every original hostile vector under RichText and TextOnly, plus exact formatting/link outputs, intermediate budgets and concurrent immutable operations. The expected strings are checked independently of the implementation under test.
