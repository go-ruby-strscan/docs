// Copyright (c) the go-ruby-strscan authors
// SPDX-License-Identifier: BSD-3-Clause
//
// Library-level benchmark driver for the pure-Go go-ruby-strscan StringScanner.
// It exercises the same StringScanner operations as ruby/strscan.rb over an
// identical, deterministic corpus, so the ns/op numbers compare the pure-Go
// library primitive against each Ruby runtime's own C-extension strscan.
//
// With CHECK=1 it instead prints one "CHECK\t<label>\t<value>" line per op: an
// integer checksum of the op's result, used to prove the Go output is
// byte-identical to MRI before any timing is trusted.
package main

import (
	"fmt"
	"os"
	"strings"

	strscan "github.com/go-ruby-strscan/strscan"
)

// corpus is a fixed, lexer-shaped input: a short arithmetic-like sentence
// repeated so a full tokenizing pass does real work. It is byte-for-byte the
// same string the Ruby workload builds.
var corpus = strings.Repeat("foo123 + bar456 - baz789 * qux000 / quux ; ", 64)

// The four token patterns a StringScanner lexer reuses over and over.
const (
	patIdent = `[A-Za-z_][A-Za-z0-9_]*`
	patNum   = `[0-9]+`
	patWS    = `\s+`
	patOp    = `[-+*/;]`
	patWord  = `[A-Za-z0-9_]+`
)

// opScan tokenizes the whole corpus with anchored Scan per token (the classic
// StringScanner lexer loop); checksum = number of tokens consumed.
func opScan() int {
	s := strscan.New(corpus)
	n := 0
	for !s.EOS() {
		matched := false
		for _, p := range []string{patIdent, patNum, patWS, patOp} {
			if _, ok := s.Scan(p); ok {
				n++
				matched = true
				break
			}
		}
		if !matched {
			s.Getch()
		}
	}
	return n
}

// opSkip walks the corpus with Skip, alternating whitespace and non-whitespace
// runs; checksum = total bytes skipped (equals len(corpus)).
func opSkip() int {
	s := strscan.New(corpus)
	total := 0
	for !s.EOS() {
		if k, ok := s.Skip(patWS); ok {
			total += k
		} else if k, ok := s.Skip(`\S+`); ok {
			total += k
		} else {
			s.Getch()
		}
	}
	return total
}

// opMatch calls match? (anchored, non-advancing) at every character position,
// advancing one char with Getch each step; checksum = sum of matched lengths.
func opMatch() int {
	s := strscan.New(corpus)
	n := 0
	for !s.EOS() {
		if m, ok := s.Match(patWord); ok {
			n += m
		}
		s.Getch()
	}
	return n
}

// opScanUntil repeatedly advances to and past each ';' with scan_until;
// checksum = total bytes returned across all hops.
func opScanUntil() int {
	s := strscan.New(corpus)
	total := 0
	for {
		r, ok := s.ScanUntil(patOp)
		if !ok {
			break
		}
		total += len(r)
	}
	return total
}

// opGetch consumes the corpus one character at a time; checksum = total bytes
// returned (equals len(corpus) for this ASCII input).
func opGetch() int {
	s := strscan.New(corpus)
	total := 0
	for !s.EOS() {
		c, ok := s.Getch()
		if !ok {
			break
		}
		total += len(c)
	}
	return total
}

// opPeek peeks a 4-byte window at each position, advancing with Getch;
// checksum = total bytes returned by peek.
func opPeek() int {
	s := strscan.New(corpus)
	total := 0
	for !s.EOS() {
		total += len(s.Peek(4))
		s.Getch()
	}
	return total
}

// ops is the ordered op table shared by the timing and CHECK paths.
var ops = []struct {
	label string
	fn    func() int
}{
	{"scan-tokenize", opScan},
	{"skip", opSkip},
	{"match?", opMatch},
	{"scan_until", opScanUntil},
	{"getch", opGetch},
	{"peek", opPeek},
}

func main() {
	if os.Getenv("CHECK") != "" {
		for _, o := range ops {
			fmt.Printf("CHECK\t%s\t%d\n", o.label, o.fn())
		}
		return
	}
	const inner = 200
	for _, o := range ops {
		fn := o.fn
		bench(o.label, inner, func() { sink = fn() })
	}
}
