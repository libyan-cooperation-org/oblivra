package oql

import "fmt"

type TokenType int

const (
	TokEOF TokenType = iota
	TokIdent
	TokString
	TokNumber
	TokPipe
	TokLParen
	TokRParen
	TokLBracket
	TokRBracket
	TokComma
	TokDot
	TokEq
	TokNeq
	TokGt
	TokGte
	TokLt
	TokLte
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokQuestion
	TokColon
	TokBang
	TokAmpAmp
	TokPipePipe
)

type Token struct {
	Type TokenType
	Val  string
	Pos  int
	Line int
	Col  int
}

func (t Token) String() string {
	if t.Type == TokEOF {
		return "EOF"
	}
	return fmt.Sprintf("%s(%q)@%d", tokenName(t.Type), t.Val, t.Pos)
}

func tokenName(t TokenType) string {
	names := map[TokenType]string{
		TokEOF: "EOF", TokIdent: "IDENT", TokString: "STRING",
		TokNumber: "NUMBER", TokPipe: "PIPE", TokLParen: "LPAREN",
		TokRParen: "RPAREN", TokLBracket: "LBRACKET", TokRBracket: "RBRACKET",
		TokComma: "COMMA", TokDot: "DOT", TokEq: "EQ", TokNeq: "NEQ",
		TokGt: "GT", TokGte: "GTE", TokLt: "LT", TokLte: "LTE",
		TokPlus: "PLUS", TokMinus: "MINUS", TokStar: "STAR",
		TokSlash: "SLASH", TokPercent: "PERCENT", TokQuestion: "QUESTION",
		TokColon: "COLON", TokBang: "BANG", TokAmpAmp: "AND", TokPipePipe: "OR",
	}
	if n, ok := names[t]; ok {
		return n
	}
	return "UNKNOWN"
}
