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

| Op | go-ruby (pure Go) | MRI | MRI + YJIT | JRuby | TruffleRuby | **go vs YJIT** |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `getch` | **5 462** | 135 745 | 99 825 | 39 553 | 69 909 | **18.3× faster** ✅ |
| `peek` | **6 224** | 241 470 | 183 415 | 60 559 | 55 523 | **29.5× faster** ✅ |
| `scan-tokenize` | 589 349 | 461 880 | 325 675 | 395 338 | 59 196 | 1.81× slower |
| `scan_until` | 49 154 | 26 350 | 21 290 | 16 198 | 8 904 | 2.31× slower |
| `match?` | 656 964 | 326 170 | 259 075 | 167 918 | 105 348 | 2.54× slower |
| `skip` | 413 573 | 132 350 | 99 375 | 85 782 | 29 827 | 4.16× slower |

## The go-vs-YJIT verdict, per op

**Beats YJIT (the pure-Go primitives that never enter the regexp engine):**

- **`getch` — 18.3× faster than YJIT** (5 462 ns vs 99 825 ns).
- **`peek` — 29.5× faster than YJIT** (6 224 ns vs 183 415 ns).

These are pure byte/rune cursor moves; the Go implementation is a slice reslice
plus a UTF-8 decode, so it leaves every interpreter — YJIT included — far behind.

**Loses to YJIT (the regexp-driven ops):**

- **`scan-tokenize` — 1.81× slower** than YJIT.
- **`scan_until` — 2.31× slower** than YJIT.
- **`match?` — 2.54× slower** than YJIT.
- **`skip` — 4.16× slower** than YJIT.

Every op that loses is dominated by regular-expression matching, which this
library delegates to the sibling pure-Go Onigmo engine
([`go-ruby-regexp`](https://github.com/go-ruby-regexp/regexp)). MRI's `strscan`
calls C Onigmo directly, and YJIT additionally removes interpreter dispatch, so
the regexp-bound ops trail by 1.8×–4.2×. This is a **regexp-engine** gap, not a
`strscan` gap — closing it is tracked in `go-ruby-regexp`, and it is the top
optimization lever for the regexp-driven `StringScanner` ops here. Output is
byte-identical to MRI on every op; only throughput on the regexp ops lags.

Net: **2 of the 6 operations beat MRI + YJIT (both decisively), 4 do not.**

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
