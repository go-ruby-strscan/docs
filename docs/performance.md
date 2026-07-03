# Performance

`go-ruby-strscan/strscan` is the pure-Go, CGO-free library that
[`rbgo`](https://github.com/go-embedded-ruby/ruby) binds for Ruby's `strscan`.
This page records a **real, library-level** benchmark of that module's Go API
against every reference runtime's own C-extension `StringScanner`, one row per
`StringScanner` operation. It is part of the ecosystem-wide per-module parity
suite, and **the bar is beating MRI + YJIT**, not just plain MRI.

## What is measured

Six representative `StringScanner` operations run over one **fixed,
deterministic corpus** (a 2 752-byte lexer-shaped input, `"foo123 + bar456 -
baz789 * qux000 / quux ; "` × 64):

| Op | What it exercises |
| --- | --- |
| `scan-tokenize` | the classic lexer loop: anchored `scan(/…/)` per token until `eos?` (4 reused patterns) |
| `skip` | `skip(/…/)` alternating whitespace / non-whitespace runs |
| `match?` | anchored, non-advancing `match?(/…/)` at every character position |
| `scan_until` | forward `scan_until(/…/)` hopping to and past each operator |
| `getch` | consume the corpus one character at a time |
| `peek` | `peek(4)` window at each position |

The **go-ruby** column drives this pure-Go library through its Go API; every
other column is that interpreter's own stdlib `strscan` C extension. The Go and
Ruby drivers build the **identical** corpus and, before any timing, each op's
integer checksum is verified **byte-identical to MRI** — all four runtimes and
the Go driver agree on every op (e.g. `scan-tokenize`=1280 tokens,
`match?`=6016, `peek`=11002). So the comparison is the same observable operation,
apples-to-apples.

- **Host:** Apple M4 Max, macOS (`arm64-darwin`). **Date:** 2026-07-03.
- **Runtimes:** Go 1.26.4; `ruby 4.0.5 +PRISM` (MRI, the oracle) and
  `ruby --yjit`; `jruby 10.1.0.0` (OpenJDK 25); `truffleruby 34.0.1`
  (GraalVM CE Native).
- **Method:** each process runs 3 untimed warm-up passes then 25 timed passes of
  a fixed inner loop, timed with a monotonic clock; the **best** pass is reported
  as **ns/op**. Interpreter start-up is outside the timed region, so the number
  is the operation's own cost, not `ruby file.rb` process cost. Numbers were
  stable to within ~1 % across repeated runs.
- Harness and drivers live in this repo under
  [`benchmarks/`](https://github.com/go-ruby-strscan/docs/tree/main/benchmarks)
  (`go/`, `ruby/strscan.rb`, `run.sh`). Reproduce: `bash benchmarks/run.sh`.

## Results (ns/op, best of 25)

These are the numbers **after** wiring the scanner onto `go-ruby-regexp`'s
allocation-free **bounds-only** match API (`MatchBoundsAt` / `MatchBounds`):
the regexp-driven ops (`scan` / `skip` / `match?` / `scan_until`) take the
whole-match span with no `MatchData` allocation, and the capture-bearing
`MatchData` is rebuilt lazily only if the caller actually reads a group
(`StringScanner#[]`) — which a tokenizing lexer never does. Output stays
byte-identical to MRI on every op, captures included.

| Op | go-ruby (pure Go) | MRI | MRI + YJIT | JRuby | TruffleRuby | **go vs YJIT** |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `peek` | **6 660** | 245 280 | 188 745 | 63 727 | 56 332 | **28.3× faster** ✅ |
| `getch` | **5 275** | 138 130 | 100 025 | 40 247 | 69 087 | **19.0× faster** ✅ |
| `scan-tokenize` | **243 787** | 458 135 | 329 090 | 389 162 | 58 185 | **1.35× faster** ✅ |
| `scan_until` | 23 344 | 27 875 | 21 600 | 16 349 | 9 425 | 1.08× slower |
| `match?` | 304 227 | 330 755 | 263 595 | 168 658 | 105 578 | 1.15× slower |
| `skip` | 163 842 | 132 395 | 99 835 | 86 048 | 30 333 | 1.64× slower |

### Before → after (this optimization)

Same host, same session, only the `go-ruby-regexp` pin + the bounds-only rewire
changed. The regexp-driven ops improved across the board; `scan-tokenize`
crossed from *behind* YJIT to *ahead* of it:

| Op | go vs YJIT before | go vs YJIT after |
| --- | ---: | ---: |
| `scan-tokenize` | 1.64× slower | **1.35× faster** ✅ |
| `scan_until` | 2.24× slower | 1.08× slower (≈ parity) |
| `match?` | 2.44× slower | 1.15× slower (beats plain MRI, 0.92×) |
| `skip` | 4.15× slower | 1.64× slower |

## The go-vs-YJIT verdict, per op

**Beats YJIT:**

- **`peek` — 28.3× faster than YJIT** (6 660 ns vs 188 745 ns).
- **`getch` — 19.0× faster than YJIT** (5 275 ns vs 100 025 ns).
- **`scan-tokenize` — 1.35× faster than YJIT** (243 787 ns vs 329 090 ns).

`peek` and `getch` are pure byte/rune cursor moves — a slice reslice plus a
UTF-8 decode — so they leave every interpreter far behind. `scan-tokenize`, the
classic anchored-`scan` lexer loop, now also **beats YJIT**: the bounds-only
anchored match runs on the regexp engine's pooled, per-token-allocation-free
lazy-NFA path, so the hot loop no longer allocates a capture array per token.

**Still behind YJIT (but much closer):**

- **`scan_until` — 1.08× slower** than YJIT (essentially parity; **beats plain
  MRI at 0.84×**).
- **`match?` — 1.15× slower** than YJIT (**beats plain MRI at 0.92×**).
- **`skip` — 1.64× slower** than YJIT.

These remain dominated by regular-expression matching, which the library
delegates to the sibling pure-Go Onigmo engine
([`go-ruby-regexp`](https://github.com/go-ruby-regexp/regexp)). MRI's `strscan`
calls C Onigmo directly and YJIT additionally removes interpreter dispatch, so a
residual gap remains on the ops whose per-call cost is a larger share regexp
setup than raw matching; `skip` (many short alternating matches) is the most
setup-bound and so trails most. Closing the rest is tracked in `go-ruby-regexp`.
Output is byte-identical to MRI on every op; only throughput on these three lags.

Net: **3 of the 6 operations now beat MRI + YJIT** (`peek`, `getch`,
`scan-tokenize`) — up from 2 — and two of the remaining three (`scan_until`,
`match?`) beat **plain MRI** and sit within ~15 % of YJIT.

!!! note "Cold-JIT caveat"
    JRuby and TruffleRuby are measured **in-process after 3 warm-up passes**, but
    3 passes is not full JIT steady state for the JVM / GraalVM — their columns
    still carry cold-to-warming-JIT cost and should be read as
    order-of-magnitude, not as fully-warmed peak throughput. TruffleRuby's very
    low `scan-tokenize` in particular reflects aggressive Graal specialization of
    that tight loop. The **go-vs-MRI and go-vs-YJIT comparisons are the load-
    bearing ones** — MRI and YJIT reach steady state within the warm-up, and both
    they and the Go driver are timed identically. All numbers are real,
    single-host measurements from the 2026-07-03 run; nothing is cherry-picked.
