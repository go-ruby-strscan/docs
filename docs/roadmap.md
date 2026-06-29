# Roadmap

`go-ruby-strscan/strscan` is grown **test-first**, each capability differential-tested against MRI
rather than built in isolation. Ruby's stringscanner — the
deterministic, interpreter-independent slice extracted from rbgo's internals — is
**complete**.

| Stage | What | Status |
| --- | --- | --- |
| Cursor model | The byte cursor `pos` and the codepoint cursor `charpos`, the scanned/pre-match/post-match regions, and `reset` / `terminate` — the StringScanner state MRI exposes. | **Done** |
| Anchored scanning | `scan`, `skip`, `match?` and `check` match a pattern **anchored at the cursor**; `scan` and `skip` advance, `match?` and `check` only report — exactly MRI's advance/no-advance split. | **Done** |
| Searching forward | `scan_until`, `skip_until`, `check_until` and `exist?` search forward for the pattern, returning (and optionally consuming) everything up to and including the match, as MRI does. | **Done** |
| Peeking & getch | `peek(n)` returns the next n bytes without advancing, `getch` consumes one (multibyte-aware) character, and `eos?` reports end-of-string. | **Done** |
| Captures & unscan | After a match, `[]` and `captures` expose the regexp groups (`[0]` the whole match, named groups included), and `unscan` rolls the cursor back to before the last scan — matching MRI's semantics. | **Done** |
| Differential oracle & coverage | A wide scan-sequence corpus run both here and by the system `ruby`, compared byte-for-byte; 100% coverage, gofmt + go vet clean, green across all six 64-bit Go arches and three OSes. Pattern matching is backed by go-ruby-regexp. | **Done** |

## Documented out-of-scope boundaries

These are **deliberate**, recorded so the module's surface is unambiguous:

- **No interpreter.** The library implements the deterministic algorithm; it
  never runs arbitrary Ruby. Anything that needs a live binding or evaluation is
  the consumer's job — that is why `rbgo` binds this module rather than the
  reverse.
- **Reference is reference Ruby (MRI).** Byte-for-byte conformance targets MRI's
  behaviour; differences across MRI releases are matched to the reference used by
  the differential oracle.
- **Standalone & reusable.** The module has no dependency on the Ruby runtime;
  the dependency runs the other way.

See [Usage & API](api.md) for the surface and [Why pure Go](why.md) for the
deterministic/interpreter split.
