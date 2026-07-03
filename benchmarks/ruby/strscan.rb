# frozen_string_literal: true
# Copyright (c) the go-ruby-strscan authors
# SPDX-License-Identifier: BSD-3-Clause
#
# Reference StringScanner workload, mirroring benchmarks/go/main.go op-for-op
# over an identical, deterministic corpus. Run normally it reports ns/op per op
# through the shared harness; run with CHECK=1 it prints one
# "CHECK\t<label>\t<value>" line per op so the Go output can be proven identical
# to MRI (the oracle) before any timing is trusted.
require "strscan"
require_relative "_harness"

# Byte-for-byte the same input the Go driver builds.
CORPUS = ("foo123 + bar456 - baz789 * qux000 / quux ; " * 64).b

PAT_IDENT = /[A-Za-z_][A-Za-z0-9_]*/
PAT_NUM   = /[0-9]+/
PAT_WS    = /\s+/
PAT_OP    = %r{[-+*/;]}
PAT_WORD  = /[A-Za-z0-9_]+/
PAT_NONWS = /\S+/

# scan: the classic StringScanner lexer loop; checksum = tokens consumed.
def op_scan
  s = StringScanner.new(CORPUS)
  n = 0
  until s.eos?
    matched = false
    [PAT_IDENT, PAT_NUM, PAT_WS, PAT_OP].each do |p|
      if s.scan(p)
        n += 1
        matched = true
        break
      end
    end
    s.getch unless matched
  end
  n
end

# skip: alternate whitespace / non-whitespace runs; checksum = bytes skipped.
def op_skip
  s = StringScanner.new(CORPUS)
  total = 0
  until s.eos?
    if (k = s.skip(PAT_WS))
      total += k
    elsif (k = s.skip(PAT_NONWS))
      total += k
    else
      s.getch
    end
  end
  total
end

# match?: anchored, non-advancing, at every char position; checksum = sum len.
def op_match
  s = StringScanner.new(CORPUS)
  n = 0
  until s.eos?
    m = s.match?(PAT_WORD)
    n += m if m
    s.getch
  end
  n
end

# scan_until: hop to and past each operator; checksum = total bytes returned.
def op_scan_until
  s = StringScanner.new(CORPUS)
  total = 0
  while (r = s.scan_until(PAT_OP))
    total += r.bytesize
  end
  total
end

# getch: consume one char at a time; checksum = total bytes returned.
def op_getch
  s = StringScanner.new(CORPUS)
  total = 0
  while (c = s.getch)
    total += c.bytesize
  end
  total
end

# peek: 4-byte window at each position, advancing with getch; checksum = bytes.
def op_peek
  s = StringScanner.new(CORPUS)
  total = 0
  until s.eos?
    total += s.peek(4).bytesize
    s.getch
  end
  total
end

OPS = [
  ["scan-tokenize", method(:op_scan)],
  ["skip",          method(:op_skip)],
  ["match?",        method(:op_match)],
  ["scan_until",    method(:op_scan_until)],
  ["getch",         method(:op_getch)],
  ["peek",          method(:op_peek)],
].freeze

if ENV["CHECK"] && !ENV["CHECK"].empty?
  OPS.each { |label, m| printf("CHECK\t%s\t%d\n", label, m.call) }
else
  INNER = 200
  OPS.each { |label, m| bench(label, INNER) { m.call } }
end
