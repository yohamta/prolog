package engine

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer_Token(t *testing.T) {
	t.Run("invalid init rune", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("\000")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenInvalid, Val: "\000"}, token)
	})

	t.Run("clause", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("append(nil,L,L).")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "append"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "nil"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "L"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "L"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("conjunction", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("p(X, Y), p(Y, X).")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "p"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "X"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "Y"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token) // operator

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "p"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "Y"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "X"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("nil", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("[]")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenOpenList, Val: "["}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseList, Val: "]"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("empty block", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("{}")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCurly, Val: "{"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseCurly, Val: "}"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("list", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("[a, b|c]")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenOpenList, Val: "["}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "a"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "b"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenHTSep, Val: "|"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "c"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseList, Val: "]"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("block", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("{a, b, c}")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCurly, Val: "{"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "a"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "b"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "c"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseCurly, Val: "}"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("comma", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader(",(x, y), p(x,,), q((,)).")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "x"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "y"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "p"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "x"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "q"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("period", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("X = .. .")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "X"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: "="}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: ".."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)
	})

	t.Run("single line comment", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("% comment\nfoo.")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "foo"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("multi line comment", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader("/* comment \n * also comment \n */foo.")), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "foo"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("insufficient in body", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader("/* comment ")), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})
	})

	t.Run("quoted", func(t *testing.T) {
		t.Run("no escape", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'abc'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'abc'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("double single quotes", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'don''t panic'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'don''t panic'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backslash at the very end of the line", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'this is \
a quoted ident'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'this is \\\na quoted ident'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("alert", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\a'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\a'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backspace", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\b'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\b'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("formfeed", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\f'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\f'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("newline", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\n'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\n'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("return", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\r'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\r'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("tab", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\t'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\t'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("vertical tab", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\v'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\v'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("hex code", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\xa3\'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\xa3\\'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("oct code", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\43\'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\43\\'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backslash", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\\'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: `'\\'`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("single quote", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\''.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: `'\''`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("double quote", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\"'.`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: `'\"'`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backquote", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader("'\\`'.")), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenQuoted, Val: "'\\`'"}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("insufficient", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'`)), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})

		t.Run("insufficient after slash", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\`)), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})

		t.Run("insufficient after hex", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\x`)), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})

		t.Run("unexpected rune after hex", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\xG`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInvalid, Val: `'\xG`}, token)
		})

		t.Run("insufficient after octal", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\0`)), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})

		t.Run("unexpected rune after octal", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`'\08`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInvalid, Val: `'\08`}, token)
		})
	})

	t.Run("integer", func(t *testing.T) {
		t.Run("decimal", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`012345`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInteger, Val: "012345"}, token)
		})

		t.Run("octal", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`0o567`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInteger, Val: "0o567"}, token)
		})

		t.Run("hexadecimal", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`0x89ABC`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInteger, Val: "0x89ABC"}, token)
		})

		t.Run("binary", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`0b10110101`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInteger, Val: "0b10110101"}, token)
		})

		t.Run("character", func(t *testing.T) {
			t.Run("normal", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`0'a`)), nil)
				token := l.Token()
				assert.Equal(t, Token{Kind: TokenInteger, Val: "0'a"}, token)
			})

			t.Run("quote", func(t *testing.T) {
				t.Run("ok", func(t *testing.T) {
					l := NewLexer(bufio.NewReader(strings.NewReader(`0'''`)), nil)
					token := l.Token()
					assert.Equal(t, Token{Kind: TokenInteger, Val: "0'''"}, token)
				})

				t.Run("insufficient", func(t *testing.T) {
					l := NewLexer(bufio.NewReader(strings.NewReader(`0'`)), nil)
					token := l.Token()
					assert.Equal(t, TokenInsufficient, token.Kind)
				})
			})

			t.Run("escape sequence", func(t *testing.T) {
				t.Run("ok", func(t *testing.T) {
					l := NewLexer(bufio.NewReader(strings.NewReader(`0'\n`)), nil)
					token := l.Token()
					assert.Equal(t, Token{Kind: TokenInteger, Val: `0'\n`}, token)
				})

				t.Run("insufficient", func(t *testing.T) {
					l := NewLexer(bufio.NewReader(strings.NewReader(`0'\`)), nil)
					token := l.Token()
					assert.Equal(t, TokenInsufficient, token.Kind)
				})

				t.Run("unknown", func(t *testing.T) {
					l := NewLexer(bufio.NewReader(strings.NewReader(`0'\q`)), nil)
					token := l.Token()
					assert.Equal(t, Token{Kind: TokenInvalid, Val: `0'\q`}, token)
				})

				t.Run("really big", func(t *testing.T) {
					l := NewLexer(bufio.NewReader(strings.NewReader(`0'\😀`)), nil)
					token := l.Token()
					assert.Equal(t, Token{Kind: TokenInvalid, Val: `0'\😀`}, token)
				})
			})
		})

		t.Run("misc", func(t *testing.T) {
			t.Run("just 0", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`0`)), nil)
				token := l.Token()
				assert.Equal(t, Token{Kind: TokenInteger, Val: "0"}, token)
			})

			t.Run("caret", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`A^foo`)), nil)
				token := l.Token()
				assert.Equal(t, Token{Kind: TokenVariable, Val: `A`}, token)
				token = l.Token()
				assert.Equal(t, Token{Kind: TokenGraphic, Val: `^`}, token)
				token = l.Token()
				assert.Equal(t, Token{Kind: TokenLetterDigit, Val: `foo`}, token)
			})
		})
	})

	t.Run("float", func(t *testing.T) {
		t.Run("without e", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`2.34`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenFloatNumber, Val: "2.34"}, token)
		})

		t.Run("with e and no sign", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`2.34E5`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenFloatNumber, Val: "2.34E5"}, token)
		})

		t.Run("with e and plus", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`2.34E+5`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenFloatNumber, Val: "2.34E+5"}, token)
		})

		t.Run("with e and minus", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`2.34E-10`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenFloatNumber, Val: "2.34E-10"}, token)
		})

		t.Run("with e and insufficient", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`2.34E`)), nil)
			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})

		t.Run("with e and unexpected rune", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`2.34E*`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenInvalid, Val: `2.34E*`}, token)
		})

		t.Run("begins with 0", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`0.333`)), nil)
			token := l.Token()
			assert.Equal(t, Token{Kind: TokenFloatNumber, Val: "0.333"}, token)
		})
	})

	t.Run("integer then period", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("X is 1 + 2.")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "X"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "is"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenInteger, Val: "1"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: "+"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenInteger, Val: "2"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEOF}, token)
	})

	t.Run("multiple terms", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader(`
foo(a).
foo(b).
foo(c).
`)), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "foo"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "a"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "foo"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "b"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "foo"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "c"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)
	})

	t.Run("char conversions", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader(`abc('abc').`)), map[rune]rune{
			'b': 'a',
		})

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "aac"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenQuoted, Val: "'abc'"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)
	})

	t.Run("trailing space in args", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader(`:- op( 20, xfx, <-- ).`)), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: ":-"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "op"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenInteger, Val: "20"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "xfx"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: "<--"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)
	})

	t.Run("elems", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader(`[V<--Ans].`)), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenOpenList, Val: "["}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "V"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: "<--"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "Ans"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseList, Val: "]"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)
	})

	t.Run("trailing space in list elems", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader(`del_item(Item, [It |R], R) :-
      same_subst(Item, It), ! .`)), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "del_item"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "Item"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenList, Val: "["}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "It"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenHTSep, Val: "|"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "R"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseList, Val: "]"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "R"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: ":-"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenLetterDigit, Val: "same_subst"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenOpenCT, Val: "("}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "Item"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenVariable, Val: "It"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenClose, Val: ")"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCut, Val: "!"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)
	})

	t.Run("double quoted", func(t *testing.T) {
		t.Run("no escape", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"abc".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"abc"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("double double quotes", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"don""t panic".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"don""t panic"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backslash at the very end of the line", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"this is \
a quoted ident".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"this is \
a quoted ident"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("alert", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\a".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\a"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backspace", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\b".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\b"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("formfeed", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\f".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\f"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("newline", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\n".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\n"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("return", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\r".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\r"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("tab", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\t".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\t"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("vertical tab", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\v".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\v"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("hex code", func(t *testing.T) {
			t.Run("ok", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`"\xa3\".`)), nil)

				token := l.Token()
				assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\xa3\"`}, token)

				token = l.Token()
				assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

				token = l.Token()
				assert.Equal(t, Token{Kind: TokenEOF}, token)
			})

			t.Run("insufficient", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`"\xa3`)), nil)

				token := l.Token()
				assert.Equal(t, TokenInsufficient, token.Kind)
			})

			t.Run("not hex", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`"\xa3g`)), nil)

				token := l.Token()
				assert.Equal(t, Token{Kind: TokenInvalid, Val: `"\xa3g`}, token)
			})
		})

		t.Run("oct code", func(t *testing.T) {
			t.Run("ok", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`"\43\".`)), nil)

				token := l.Token()
				assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\43\"`}, token)

				token = l.Token()
				assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

				token = l.Token()
				assert.Equal(t, Token{Kind: TokenEOF}, token)
			})

			t.Run("insufficient", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`"\43`)), nil)

				token := l.Token()
				assert.Equal(t, TokenInsufficient, token.Kind)
			})

			t.Run("not octal", func(t *testing.T) {
				l := NewLexer(bufio.NewReader(strings.NewReader(`"\438`)), nil)

				token := l.Token()
				assert.Equal(t, Token{Kind: TokenInvalid, Val: `"\438`}, token)
			})
		})

		t.Run("backslash", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\\".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\\"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("single quote", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\'".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\'"`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("double quote", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\"".`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: `"\""`}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("backquote", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader("\"\\`\".")), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenDoubleQuotedList, Val: "\"\\`\""}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEnd, Val: "."}, token)

			token = l.Token()
			assert.Equal(t, Token{Kind: TokenEOF}, token)
		})

		t.Run("insufficient", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"`)), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})

		t.Run("escape then insufficient", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`"\`)), nil)

			token := l.Token()
			assert.Equal(t, TokenInsufficient, token.Kind)
		})
	})

	t.Run("graphic token begins with sign", func(t *testing.T) {
		t.Run("plus", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`++`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenGraphic, Val: `++`}, token)
		})

		t.Run("minus", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader(`--`)), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenGraphic, Val: `--`}, token)
		})
	})

	t.Run("graphic token begins with slash", func(t *testing.T) {
		t.Run("single", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader("/")), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenGraphic, Val: "/"}, token)
		})

		t.Run("multi", func(t *testing.T) {
			l := NewLexer(bufio.NewReader(strings.NewReader("//")), nil)

			token := l.Token()
			assert.Equal(t, Token{Kind: TokenGraphic, Val: "//"}, token)
		})
	})

	t.Run("period in list", func(t *testing.T) {
		l := NewLexer(bufio.NewReader(strings.NewReader("['1',.,'2']")), nil)

		token := l.Token()
		assert.Equal(t, Token{Kind: TokenOpenList, Val: "["}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenQuoted, Val: "'1'"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenGraphic, Val: "."}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenComma, Val: ","}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenQuoted, Val: "'2'"}, token)

		token = l.Token()
		assert.Equal(t, Token{Kind: TokenCloseList, Val: "]"}, token)
	})
}
