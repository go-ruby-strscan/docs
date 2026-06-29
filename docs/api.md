# Usage & API

The public API lives at the module root (`github.com/go-ruby-strscan/strscan`). It is **Ruby-shaped but Go-idiomatic**: a `Scanner` value carries the cursor and exposes the StringScanner method surface; the surface follows Go conventions — explicit returns, value types, no global state. Pattern matching uses [go-ruby-regexp](https://github.com/go-ruby-regexp).

!!! success "Status: implemented"
    The library is built and importable as `github.com/go-ruby-strscan/strscan`, bound into
    `rbgo` as a native module; see [Roadmap](roadmap.md).

## Install

```sh
go get github.com/go-ruby-strscan/strscan
```

## Worked example

```go
sc := strscan.New("3 + 41 = 44")
n, _  := sc.Scan(`\d+`)        // "3",  cursor after "3"
_      = sc.Skip(`\s*\+\s*`)   // skip " + "
m, _  := sc.Scan(`\d+`)        // "41"
rest  := sc.Peek(10)           // " = 44" without advancing
_      = n; _ = m; _ = rest
```

## Shape

```go
// Scanner is a cursor over a string, advanced by matching patterns.
type Scanner struct{ /* ... */ }

func New(s string) *Scanner

func (s *Scanner) Scan(pattern string) (string, bool)        // anchored, advances
func (s *Scanner) ScanUntil(pattern string) (string, bool)   // searches forward
func (s *Scanner) Skip(pattern string) (int, bool)           // anchored, length
func (s *Scanner) Match(pattern string) (int, bool)          // match? — no advance
func (s *Scanner) Check(pattern string) (string, bool)       // check — no advance
func (s *Scanner) Peek(n int) string
func (s *Scanner) Getch() (string, bool)
func (s *Scanner) Pos() int                                  // byte cursor
func (s *Scanner) CharPos() int                              // codepoint cursor
func (s *Scanner) Captures() []string                        // last match's groups
func (s *Scanner) Unscan() error                             // undo the last scan
```

## MRI conformance

Correctness is defined by reference Ruby. A **differential oracle** runs a wide
corpus through both the system `ruby` and this library and compares the results
**byte-for-byte** — not approximated from memory. The oracle tests skip
themselves where `ruby` is not on `PATH` (e.g. the qemu arch lanes), so the
cross-arch builds still validate the library.

## Relationship to Ruby

`go-ruby-strscan/strscan` is **standalone and reusable**, and is the backend bound into
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) by `rbgo` as a
native module — the same way [go-ruby-regexp](https://github.com/go-ruby-regexp)
and [go-ruby-erb](https://github.com/go-ruby-erb) are bound. The dependency runs
the other way: this library has no dependency on the Ruby runtime.
