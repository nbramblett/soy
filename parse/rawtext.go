package parse

import "unicode/utf8"

type rawtextlexer struct {
	str      string
	pos      int
	lastpos  int
	lastpos2 int
}

func (l *rawtextlexer) eof() bool {
	return l.pos >= len(l.str)
}
func (l *rawtextlexer) next() rune {
	l.lastpos2 = l.lastpos
	l.lastpos = l.pos
	var r, width = utf8.DecodeRuneInString(l.str[l.pos:])
	l.pos += width
	return r
}
func (l *rawtextlexer) backup() {
	l.pos = l.lastpos
	l.lastpos = l.lastpos2
	l.lastpos2 = 0
}
func (l *rawtextlexer) emitRune(result []byte) []byte {
	return append(result, []byte(l.str[l.lastpos:l.pos])...)
}

// rawtext processes the raw text found in templates:
// - strip comments (// to end of line)
// - trim leading/trailing whitespace if trimPrefix/trimSuffix are true.
// - trim leading and trailing whitespace on each internal line
// - join lines with no space if '<' or '>' are on either side, else with 1 space.
// - everywhere, collapse multiple spaces to single space
// - trim leading/trailing space only if there is a newline before/after anything else.
func rawtext(s string, trimPrefix, trimSuffix bool) []byte {
	var lex = rawtextlexer{s, 0, 0, 0}
	var (
		trimming       = false
		seenNewline    = false
		lastChar       rune
		charBeforeTrim rune
		result         = make([]byte, 0, len(s))
	)
	for {
		if lex.eof() {
			// add a space if we've been trimming, unless either:
			// - trimSuffix == true and we've seen a newline
			// - trimPrefix == true and we're still at prefix and we've seen a newline
			if (!(seenNewline && (trimSuffix || (trimPrefix && charBeforeTrim == 0)))) && trimming {
				result = append(result, ' ')
			}
			return result
		}
		var r = lex.next()

		// comment removal
		if r == '/' {
			if lex.next() == '/' {
				for {
					r = lex.next()
					if lex.eof() {
						return result
					}
					if isEndOfLine(r) {
						break
					}
				}
			}
			lex.backup()
		}

		// collapse space / join lines
		if trimming {
			// more space, keep going
			if isSpace(r) {
				continue
			}
			if isEndOfLine(r) {
				seenNewline = true
				continue
			}

			// done with the trim. add a space if either:
			// - we haven't seen an newline
			// - the character before and after are not tight joiners
			// unless we are trimming the prefix, and trimPrefix == true, and we've seen a newline
			var isPrefix = charBeforeTrim == 0
			if !(trimPrefix && isPrefix && seenNewline) &&
				(!seenNewline || (!isTightJoiner(charBeforeTrim) && !isTightJoiner(r))) {
				result = append(result, ' ')
			}
			trimming = false
			seenNewline = false
		}

		// begin to trim
		seenNewline = isEndOfLine(r)
		if isSpace(r) || seenNewline {
			trimming = true
			charBeforeTrim = lastChar
			continue
		}

		// non-space characters are added verbatim.
		result = lex.emitRune(result)
		lastChar = r
	}
	return result
}

func isTightJoiner(r rune) bool {
	switch r {
	case '<', '>':
		return true
	}
	return false
}
