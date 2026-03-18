package oql

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func Tokenize(input string) ([]Token, error) {
	l := &lexer{input: input, line: 1, col: 1}
	return l.run()
}

type lexer struct {
	input  string
	pos    int
	line   int
	col    int
	tokens []Token
}

func (l *lexer) peekRune() (rune, int) {
	if l.pos >= len(l.input) {
		return 0, 0
	}
	return utf8.DecodeRuneInString(l.input[l.pos:])
}


func (l *lexer) peekByte() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) peekNextByte() byte {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *lexer) advByte() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *lexer) emit(t TokenType, v string, p, ln, c int) {
	l.tokens = append(l.tokens, Token{Type: t, Val: v, Pos: p, Line: ln, Col: c})
}

func (l *lexer) run() ([]Token, error) {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}
		ch := l.peekByte()
		sp, sl, sc := l.pos, l.line, l.col

		switch {
		case ch == '|':
			if l.peekNextByte() == '|' {
				l.emit(TokPipePipe, "||", sp, sl, sc)
				l.advByte()
				l.advByte()
			} else {
				l.emit(TokPipe, "|", sp, sl, sc)
				l.advByte()
			}
		case ch == '(':
			l.emit(TokLParen, "(", sp, sl, sc)
			l.advByte()
		case ch == ')':
			l.emit(TokRParen, ")", sp, sl, sc)
			l.advByte()
		case ch == '[':
			l.emit(TokLBracket, "[", sp, sl, sc)
			l.advByte()
		case ch == ']':
			l.emit(TokRBracket, "]", sp, sl, sc)
			l.advByte()
		case ch == ',':
			l.emit(TokComma, ",", sp, sl, sc)
			l.advByte()
		case ch == '?':
			l.emit(TokQuestion, "?", sp, sl, sc)
			l.advByte()
		case ch == ':':
			l.emit(TokColon, ":", sp, sl, sc)
			l.advByte()
		case ch == '+':
			l.emit(TokPlus, "+", sp, sl, sc)
			l.advByte()
		case ch == '-':
			l.emit(TokMinus, "-", sp, sl, sc)
			l.advByte()
		case ch == '*':
			l.emit(TokStar, "*", sp, sl, sc)
			l.advByte()
		case ch == '/':
			l.emit(TokSlash, "/", sp, sl, sc)
			l.advByte()
		case ch == '%':
			l.emit(TokPercent, "%", sp, sl, sc)
			l.advByte()
		case ch == '!':
			if l.peekNextByte() == '=' {
				l.emit(TokNeq, "!=", sp, sl, sc)
				l.advByte()
				l.advByte()
			} else {
				l.emit(TokBang, "!", sp, sl, sc)
				l.advByte()
			}
		case ch == '=':
			l.emit(TokEq, "=", sp, sl, sc)
			l.advByte()
		case ch == '>':
			if l.peekNextByte() == '=' {
				l.emit(TokGte, ">=", sp, sl, sc)
				l.advByte()
				l.advByte()
			} else {
				l.emit(TokGt, ">", sp, sl, sc)
				l.advByte()
			}
		case ch == '<':
			if l.peekNextByte() == '=' {
				l.emit(TokLte, "<=", sp, sl, sc)
				l.advByte()
				l.advByte()
			} else {
				l.emit(TokLt, "<", sp, sl, sc)
				l.advByte()
			}
		case ch == '&':
			if l.peekNextByte() == '&' {
				l.emit(TokAmpAmp, "&&", sp, sl, sc)
				l.advByte()
				l.advByte()
			} else {
				return nil, fmt.Errorf("unexpected '&' at line %d col %d", l.line, l.col)
			}
		case ch == '"' || ch == '\'':
			tok, err := l.readString(ch, sp, sl, sc)
			if err != nil {
				return nil, err
			}
			l.tokens = append(l.tokens, tok)
		case ch == '.' && !isDigitByte(l.peekNextByte()):
			l.emit(TokDot, ".", sp, sl, sc)
			l.advByte()
		case isDigitByte(ch) || (ch == '.' && isDigitByte(l.peekNextByte())):
			l.tokens = append(l.tokens, l.readNumber(sp, sl, sc))
		default:
			r, _ := l.peekRune()
			if isIdentStartRune(r) {
				l.tokens = append(l.tokens, l.readIdent(sp, sl, sc))
			} else {
				return nil, fmt.Errorf("unexpected character %q at line %d col %d", string(r), l.line, l.col)
			}
		}
	}
	l.tokens = append(l.tokens, Token{Type: TokEOF, Pos: l.pos, Line: l.line, Col: l.col})
	return l.tokens, nil
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.peekByte()
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advByte()
		} else if ch == '#' || (ch == '/' && l.peekNextByte() == '/') {
			for l.pos < len(l.input) && l.peekByte() != '\n' {
				l.advByte()
			}
		} else {
			break
		}
	}
}

func (l *lexer) readString(quote byte, sp, sl, sc int) (Token, error) {
	l.advByte()
	var b strings.Builder
	for l.pos < len(l.input) {
		ch := l.peekByte()
		if ch == '\\' && l.pos+1 < len(l.input) {
			l.advByte()
			switch l.peekByte() {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '\'':
				b.WriteByte('\'')
			default:
				b.WriteByte('\\')
				b.WriteByte(l.peekByte())
			}
			l.advByte()
			continue
		}
		if ch == quote {
			l.advByte()
			return Token{Type: TokString, Val: b.String(), Pos: sp, Line: sl, Col: sc}, nil
		}
		r, size := l.peekRune()
		b.WriteRune(r)
		l.pos += size
		l.col++
	}
	return Token{}, fmt.Errorf("unterminated string at line %d col %d", sl, sc)
}

func (l *lexer) readNumber(sp, sl, sc int) Token {
	start := l.pos
	hasDot := false
	hasExp := false
	for l.pos < len(l.input) {
		ch := l.peekByte()
		if isDigitByte(ch) {
			l.advByte()
		} else if ch == '.' && !hasDot && !hasExp {
			hasDot = true
			l.advByte()
		} else if (ch == 'e' || ch == 'E') && !hasExp {
			hasExp = true
			l.advByte()
			if l.pos < len(l.input) && (l.peekByte() == '+' || l.peekByte() == '-') {
				l.advByte()
			}
		} else {
			break
		}
	}
	return Token{Type: TokNumber, Val: l.input[start:l.pos], Pos: sp, Line: sl, Col: sc}
}

func (l *lexer) readIdent(sp, sl, sc int) Token {
	start := l.pos
	for l.pos < len(l.input) {
		r, size := l.peekRune()
		if isIdentPartRune(r) {
			l.pos += size
			l.col++
		} else {
			break
		}
	}
	return Token{Type: TokIdent, Val: l.input[start:l.pos], Pos: sp, Line: sl, Col: sc}
}

func isDigitByte(ch byte) bool     { return ch >= '0' && ch <= '9' }
func isIdentStartRune(r rune) bool { return unicode.IsLetter(r) || r == '_' || r == '@' }
func isIdentPartRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '*' || r == '?' || r == ':' || r == '/' || r == '@' || r == '-'
}
