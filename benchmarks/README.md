<!-- SPDX-License-Identifier: BSD-3-Clause -->
# `go-ruby-strscan` library-level benchmark harness

Reproducible, cross-runtime benchmark of the **pure-Go `go-ruby-strscan`
library** against the reference Ruby runtimes (MRI, MRI + YJIT, JRuby,
TruffleRuby). It measures each `StringScanner` **operation** through the Go API,
isolated from the rbgo interpreter, so the numbers answer: *is the pure-Go
`StringScanner` as fast as the reference runtime's own C-extension `strscan` —
and does it beat MRI + YJIT?*

## Layout

- `go/`             — self-contained Go driver; `go.mod` pins the **published**
  library by pseudo-version (no `replace`).
- `ruby/strscan.rb` — the equivalent workload; `ruby/_harness.rb` is the shared
  timer.
- `run.sh`          — runs every available runtime and prints one Markdown table
  per operation (ns/op + ratio vs MRI).

## Run

```sh
bash benchmarks/run.sh
```

Environment knobs: `OUTER` (timed passes, default 25), `WARM` (untimed warm-up
passes, default 3), and `RUBY`/`JRUBY`/`TRUFFLERUBY` to select runtime binaries.

## Method

Each process runs `WARM` untimed passes (to let the JVM/GraalVM JITs warm up),
then `OUTER` timed passes of a fixed inner loop, timed with a monotonic clock;
the **best** pass is reported as **ns/op**. Interpreter start-up is outside the
timed region. The Go driver and the Ruby script build the **identical**
deterministic corpus, and each op's integer checksum is verified byte-identical
across all runtimes (`CHECK=1 go run .` / `CHECK=1 ruby ruby/strscan.rb`) before
timing. Published, dated results are in [`../docs/performance.md`](../docs/performance.md).
