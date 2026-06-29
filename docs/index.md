# go-ruby-strscan documentation

**Ruby's `StringScanner` in pure Go — MRI-compatible, no cgo.**

`go-ruby-strscan/strscan` is a faithful, pure-Go (zero cgo) reimplementation of Ruby's StringScanner,
matching reference Ruby (MRI) byte-for-byte. The module path is
`github.com/go-ruby-strscan/strscan`.

It was **extracted from rbgo's prelude/internals into a reusable standalone
library**: the module is standalone and importable by any Go program, and it is
the backend bound into [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby)
by `rbgo` as a native module — just like
[go-ruby-regexp](https://github.com/go-ruby-regexp) and
[go-ruby-erb](https://github.com/go-ruby-erb). The dependency runs the other
way: this library has **no dependency on the Ruby runtime**.

!!! success "Status: scanner complete — MRI byte-exact"
    Faithful port of MRI's `strscan.c`: the full method surface — **`scan` / `scan_until` / `skip`**, **`match?` / `check` / `check_until`**, **`peek` / `getch`**, **`pos` / `charpos`**, **captures (`[]` / `captures`)** and **`unscan`** — with pattern matching backed by **go-ruby-regexp**. Validated by a **differential oracle** against the system `ruby` — scan results compared byte-for-byte — at 100% coverage, `gofmt` + `go vet` clean, CI green across the six 64-bit Go targets and three OSes.

## Quick taste

```go
sc := strscan.New("3 + 41 = 44")
n, _  := sc.Scan(`\d+`)        // "3",  cursor after "3"
_      = sc.Skip(`\s*\+\s*`)   // skip " + "
m, _  := sc.Scan(`\d+`)        // "41"
rest  := sc.Peek(10)           // " = 44" without advancing
_      = n; _ = m; _ = rest
```

## Repositories

| Repo | What it is |
| --- | --- |
| [`strscan`](https://github.com/go-ruby-strscan/strscan) | the library — Ruby's StringScanner in pure Go |
| [`docs`](https://github.com/go-ruby-strscan/docs) | this documentation site (MkDocs Material, versioned with mike) |
| [`go-ruby-strscan.github.io`](https://github.com/go-ruby-strscan/go-ruby-strscan.github.io) | the organization landing page (Hugo) |
| [`brand`](https://github.com/go-ruby-strscan/brand) | logo and brand assets |

## Principles

- **Pure Go, `CGO_ENABLED=0`** — trivial cross-compilation, a single static
  binary, no C toolchain.
- **MRI byte-exact.** Output matches reference Ruby exactly, not approximately,
  validated by a differential oracle against the `ruby` binary.
- **Standalone & reusable.** Extracted from rbgo's internals; no dependency on
  the Ruby runtime — the dependency runs the other way.
- **100% test coverage** is the target, enforced as a CI gate, across 6 arches
  and 3 OSes.

## Where to go next

- [Why pure Go](why.md) — why this slice of Ruby is deterministic enough to live
  as a standalone, interpreter-independent Go library.
- [Usage & API](api.md) — the public surface and worked examples.
- [Roadmap](roadmap.md) — what is done and what is downstream by design.

Source lives at [github.com/go-ruby-strscan/strscan](https://github.com/go-ruby-strscan/strscan).
