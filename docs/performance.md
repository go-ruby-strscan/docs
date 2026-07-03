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
  stable to within ~5 % across repeated runs.
- Harness and drivers live in this repo under
  [`benchmarks/`](https://github.com/go-ruby-strscan/docs/tree/main/benchmarks)
  (`go/`, `ruby/strscan.rb`, `run.sh`). Reproduce: `bash benchmarks/run.sh`.

## Results (ns/op, best of 25)

These are the numbers **after** `go-ruby-regexp` landed its anchored
**class-run consumer** — a fast path in the regexp VM that consumes a run of a
single character class (`\s+`, `\S+`, `[0-9]+`, `[A-Za-z0-9_]+`, …) directly at
the anchor with no per-byte VM dispatch. The scanner reaches it for free through
the allocation-free **bounds-only** match API (`MatchBoundsAt` / `MatchBounds`)
it already calls: no strscan code changed, only the `go-ruby-regexp` pin was
bumped. The regexp-driven ops whose pattern is a single anchored class-repeat —
`skip`, `match?`, and the class tokens inside `scan-tokenize` — take the win
directly; capture-bearing `MatchData` is still rebuilt lazily only if the caller
reads a group (`StringScanner#[]`), which a tokenizing lexer never does. Output
stays byte-identical to MRI on every op, captures included.

| Op | go-ruby (pure Go) | MRI | MRI + YJIT | JRuby | TruffleRuby | **go vs YJIT** |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `peek` | **6 666** | 245 935 | 187 195 | 60 654 | 56 231 | **28.1× faster** ✅ |
| `getch` | **5 516** | 137 450 | 99 585 | 39 627 | 68 888 | **18.1× faster** ✅ |
| `match?` | **50 972** | 330 070 | 259 040 | 166 611 | 106 811 | **5.08× faster** ✅ |
| `skip` | **34 785** | 132 420 | 96 910 | 84 370 | 29 768 | **2.79× faster** ✅ |
| `scan-tokenize` | **167 708** | 464 985 | 327 880 | 383 669 | 57 140 | **1.96× faster** ✅ |
| `scan_until` | 23 884 | 27 090 | 21 140 | 16 547 | 9 123 | 1.13× slower |

### Before → after (this optimization)

Same host, same session shape, only the `go-ruby-regexp` pin changed — from the
bounds-only match API (previous page revision) to the anchored class-run
consumer. The single-class-repeat ops moved the most; `skip` and `match?`
crossed from *behind* YJIT to well *ahead* of it:

| Op | go vs YJIT before | go vs YJIT after |
| --- | ---: | ---: |
| `skip` | 1.64× slower | **2.79× faster** ✅ |
| `match?` | 1.15× slower | **5.08× faster** ✅ |
| `scan-tokenize` | 1.35× faster | **1.96× faster** ✅ |
| `scan_until` | 1.08× slower | 1.13× slower (no class-run fast path) |
| `getch` | 19.0× faster | 18.1× faster (unchanged, non-regexp) |
| `peek` | 28.3× faster | 28.1× faster (unchanged, non-regexp) |

`skip` was the whole point: it is the most match-setup-bound op (many short
alternating `\s+` / `\S+` runs), so it gained the most once the VM stopped
dispatching per byte — from **1.64× behind** YJIT to **2.79× ahead**. `match?`
(a single anchored `[A-Za-z0-9_]+` at every position) is the same shape and
jumped from 1.15× behind to **5.08× ahead**.

## The go-vs-YJIT verdict, per op

**Beats YJIT (5 of 6):**

- **`peek` — 28.1× faster than YJIT** (6 666 ns vs 187 195 ns).
- **`getch` — 18.1× faster than YJIT** (5 516 ns vs 99 585 ns).
- **`match?` — 5.08× faster than YJIT** (50 972 ns vs 259 040 ns).
- **`skip` — 2.79× faster than YJIT** (34 785 ns vs 96 910 ns).
- **`scan-tokenize` — 1.96× faster than YJIT** (167 708 ns vs 327 880 ns).

`peek` and `getch` are pure byte/rune cursor moves — a slice reslice plus a
UTF-8 decode — so they leave every interpreter far behind. The three
regexp-driven wins (`match?`, `skip`, `scan-tokenize`) all now beat YJIT because
their hot pattern is a single anchored character-class repeat, exactly what the
new class-run consumer collapses into one tight loop over the input bytes with
no `MatchData` allocation and no per-byte VM step.

**Still behind YJIT (1 of 6, but much closer):**

- **`scan_until` — 1.13× slower** than YJIT (**beats plain MRI at 0.88×**).

`scan_until` is the one op the class-run consumer does not accelerate: it is a
*forward search* for the next operator, not an anchored class-repeat consume, so
it still walks the regexp engine's general search path. MRI's `strscan` calls C
Onigmo directly and YJIT additionally removes interpreter dispatch, leaving a
residual ~13 % gap; it still beats plain MRI. Closing the last search-path gap is
tracked in [`go-ruby-regexp`](https://github.com/go-ruby-regexp/regexp). Output
is byte-identical to MRI on every op; only `scan_until` throughput lags.

Net: **5 of the 6 operations now beat MRI + YJIT** (`peek`, `getch`, `match?`,
`skip`, `scan-tokenize`) — up from 2 — and the last one (`scan_until`) beats
**plain MRI** and sits within ~13 % of YJIT.

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
