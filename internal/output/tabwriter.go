package output

import (
	"bytes"
	"io"
	"text/tabwriter"
	"unicode/utf8"

	"golang.org/x/text/width"
)

// widthFiller is a private-use rune inserted after every East Asian wide or
// fullwidth rune before text reaches text/tabwriter, and removed again from
// tabwriter's output. tabwriter measures cells in runes, so one filler per
// wide rune makes its measurement match the terminal display width (2
// columns). Because the filler is stripped after padding is computed, it never
// reaches the terminal.
const widthFiller = '\uE000'

var widthFillerBytes = []byte(string(widthFiller))

// TabWriter is a text/tabwriter.Writer that aligns columns on terminal display
// columns instead of runes, so tables containing CJK text stay aligned.
type TabWriter struct {
	tw      *tabwriter.Writer
	pending []byte
	scratch []byte
}

// NewTabWriter returns the table writer used by all-cli's human-readable
// tables (minwidth 0, tabwidth 4, padding 2, space padding).
func NewTabWriter(w io.Writer) *TabWriter {
	return &TabWriter{
		tw: tabwriter.NewWriter(&fillerStripper{w: w}, 0, 4, 2, ' ', 0),
	}
}

// Write implements io.Writer. Incomplete UTF-8 sequences at the end of p are
// held until the next Write or Flush.
func (t *TabWriter) Write(p []byte) (int, error) {
	data := p
	if len(t.pending) > 0 {
		data = append(t.pending, p...)
		t.pending = nil
	}

	out := t.scratch[:0]
	for len(data) > 0 {
		if !utf8.FullRune(data) {
			t.pending = append([]byte(nil), data...)
			break
		}
		r, size := utf8.DecodeRune(data)
		out = append(out, data[:size]...)
		if r != utf8.RuneError && runeDisplayWidth(r) == 2 {
			out = append(out, widthFillerBytes...)
		}
		data = data[size:]
	}
	t.scratch = out

	if _, err := t.tw.Write(out); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Flush writes any buffered table to the underlying writer.
func (t *TabWriter) Flush() error {
	if len(t.pending) > 0 {
		if _, err := t.tw.Write(t.pending); err != nil {
			return err
		}
		t.pending = nil
	}
	return t.tw.Flush()
}

// runeDisplayWidth reports how many terminal columns r occupies: 2 for East
// Asian wide and fullwidth runes, 1 otherwise.
func runeDisplayWidth(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

// fillerStripper removes widthFiller runes from tabwriter output. A trailing
// partial filler sequence is held until the next Write so a split write can
// never leak half a rune.
type fillerStripper struct {
	w       io.Writer
	pending []byte
}

func (s *fillerStripper) Write(p []byte) (int, error) {
	data := p
	if len(s.pending) > 0 {
		data = append(s.pending, p...)
		s.pending = nil
	}
	if keep := partialFillerSuffix(data); keep > 0 {
		s.pending = append([]byte(nil), data[len(data)-keep:]...)
		data = data[:len(data)-keep]
	}
	if _, err := s.w.Write(bytes.ReplaceAll(data, widthFillerBytes, nil)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// partialFillerSuffix returns the length of the longest proper prefix of the
// filler encoding that data ends with.
func partialFillerSuffix(data []byte) int {
	for n := len(widthFillerBytes) - 1; n > 0; n-- {
		if bytes.HasSuffix(data, widthFillerBytes[:n]) {
			return n
		}
	}
	return 0
}
