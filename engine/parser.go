package engine

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// ErrInsufficient is a parsing error which can be resolved by adding more characters to the input.
	ErrInsufficient = errors.New("insufficient")
	errExpectation  = errors.New("expectation error")
	errNoOp         = errors.New("no op")
	errNotANumber   = errors.New("not a number")
	errPlaceholder  = errors.New("not enough arguments for placeholders")
)

// Parser turns bytes into Term.
type Parser struct {
	lexer        Lexer
	buf          tokenRingBuffer
	operators    *operators
	placeholder  Atom
	args         []Term
	doubleQuotes doubleQuotes
	vars         *[]ParsedVariable
}

// ParsedVariable is a set of information regarding a variable in a parsed term.
type ParsedVariable struct {
	Name     Atom
	Variable Variable
	Count    int
}

func newParser(input io.Reader, opts ...parserOption) *Parser {
	p := Parser{
		lexer: Lexer{
			input: newRuneRingBuffer(input),
		},
	}
	for _, o := range opts {
		o(&p)
	}
	return &p
}

type parserOption func(p *Parser)

func withCharConversions(charConversions map[rune]rune) parserOption {
	return func(p *Parser) {
		p.lexer.charConversions = charConversions
	}
}

func withOperators(operators *operators) parserOption {
	return func(p *Parser) {
		p.operators = operators
	}
}

func withDoubleQuotes(quotes doubleQuotes) parserOption {
	return func(p *Parser) {
		p.doubleQuotes = quotes
	}
}

func withParsedVars(vars *[]ParsedVariable) parserOption {
	return func(p *Parser) {
		p.vars = vars
	}
}

// Replace registers placeholder and its arguments. Every occurrence of placeholder will be replaced by arguments.
// Mismatch of the number of occurrences of placeholder and the number of arguments raises an error.
func (p *Parser) Replace(placeholder Atom, args ...interface{}) error {
	p.placeholder = placeholder
	p.args = make([]Term, len(args))
	for i, a := range args {
		var err error
		p.args[i], err = termOf(reflect.ValueOf(a))
		if err != nil {
			return err
		}
	}
	return nil
}

func termOf(o reflect.Value) (Term, error) {
	if t, ok := o.Interface().(Term); ok {
		return t, nil
	}

	switch o.Kind() {
	case reflect.Float32, reflect.Float64:
		return Float(o.Float()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Integer(o.Int()), nil
	case reflect.String:
		return Atom(o.String()), nil
	case reflect.Array, reflect.Slice:
		l := o.Len()
		es := make([]Term, l)
		for i := 0; i < l; i++ {
			var err error
			es[i], err = termOf(o.Index(i))
			if err != nil {
				return nil, err
			}
		}
		return List(es...), nil
	default:
		return nil, fmt.Errorf("can't convert to term: %v", o)
	}
}

func (p *Parser) next() Token {
	if p.buf.empty() {
		p.buf.put(p.lexer.Token())
	}
	return p.buf.get()
}

func (p *Parser) backup() {
	p.buf.backup()
}

func (p *Parser) current() Token {
	return p.buf.current()
}

// Term parses a term followed by a full stop.
func (p *Parser) Term() (Term, error) {
	if t := p.next(); t.Kind == TokenEOF {
		return nil, io.EOF
	}
	p.backup()

	t, err := p.term(1201)
	switch err {
	case nil:
		break
	case errExpectation:
		switch p.current().Kind {
		case TokenEOF, TokenInsufficient:
			return nil, ErrInsufficient
		default:
			return nil, unexpectedTokenError{actual: p.current()}
		}
	default:
		return nil, err
	}

	switch t := p.next(); t.Kind {
	case TokenEnd:
		break
	case TokenEOF, TokenInsufficient:
		return nil, ErrInsufficient
	default:
		p.backup()
		return nil, unexpectedTokenError{actual: p.current()}
	}

	if len(p.args) != 0 {
		return nil, fmt.Errorf("too many arguments for placeholders: %s", p.args)
	}

	return t, nil
}

// Number parses a number term.
func (p *Parser) Number() (Number, error) {
	var (
		n   Number
		err error
	)
	switch t := p.next(); t.Kind {
	case TokenInteger:
		n, err = parseInteger(1, t.Val)
	case TokenFloatNumber:
		n, err = parseFloat(1, t.Val)
	default:
		p.backup()
		var a Atom
		a, err = p.name()
		if err != nil {
			return nil, errNotANumber
		}

		if a != "-" {
			p.backup()
			return nil, errNotANumber
		}

		switch t := p.next(); t.Kind {
		case TokenInteger:
			n, err = parseInteger(-1, t.Val)
		case TokenFloatNumber:
			n, err = parseFloat(-1, t.Val)
		default:
			p.backup()
			p.backup()
			return nil, errNotANumber
		}
	}
	if err != nil {
		return nil, err
	}

	// No more tokens/runes after a number.
	if !p.buf.empty() || p.lexer.reserved != (Token{}) {
		return nil, errNotANumber
	}
	if r := p.lexer.rawNext(); r != utf8.RuneError {
		return nil, errNotANumber
	}

	return n, nil
}

// More checks if the parser has more tokens to read.
func (p *Parser) More() bool {
	t := p.next()
	p.backup()
	return t.Kind != TokenEOF && t.Kind != TokenInsufficient
}

type operatorSpecifier uint8

const (
	operatorSpecifierFX operatorSpecifier = iota
	operatorSpecifierFY
	operatorSpecifierXF
	operatorSpecifierYF
	operatorSpecifierXFX
	operatorSpecifierXFY
	operatorSpecifierYFX
)

func (s operatorSpecifier) term() Term {
	return [...]Term{
		operatorSpecifierFX:  Atom("fx"),
		operatorSpecifierFY:  Atom("fy"),
		operatorSpecifierXF:  Atom("xf"),
		operatorSpecifierYF:  Atom("yf"),
		operatorSpecifierXFX: Atom("xfx"),
		operatorSpecifierXFY: Atom("xfy"),
		operatorSpecifierYFX: Atom("yfx"),
	}[s]
}
func (s operatorSpecifier) arity() int {
	return [...]int{
		operatorSpecifierFX:  1,
		operatorSpecifierFY:  1,
		operatorSpecifierXF:  1,
		operatorSpecifierYF:  1,
		operatorSpecifierXFX: 2,
		operatorSpecifierXFY: 2,
		operatorSpecifierYFX: 2,
	}[s]
}

type operators []operator

type operator struct {
	priority  Integer // 1 ~ 1200
	specifier operatorSpecifier
	name      Atom
}

// Pratt parser's binding powers but in Prolog priority.
func (o *operator) bindingPriorities() (Integer, Integer) {
	const max = Integer(1202)
	switch o.specifier {
	case operatorSpecifierFX:
		return max, o.priority - 1
	case operatorSpecifierFY:
		return max, o.priority
	case operatorSpecifierXF:
		return o.priority - 1, max
	case operatorSpecifierYF:
		return o.priority, max
	case operatorSpecifierXFX:
		return o.priority - 1, o.priority - 1
	case operatorSpecifierXFY:
		return o.priority - 1, o.priority
	case operatorSpecifierYFX:
		return o.priority, o.priority - 1
	default:
		return max, max
	}
}

type doubleQuotes int

const (
	doubleQuotesChars doubleQuotes = iota
	doubleQuotesCodes
	doubleQuotesAtom
)

func (d doubleQuotes) String() string {
	return [...]string{
		doubleQuotesCodes: "codes",
		doubleQuotesChars: "chars",
		doubleQuotesAtom:  "atom",
	}[d]
}

// Loosely based on Pratt parser explained in this article: https://matklad.github.io/2020/04/13/simple-but-powerful-pratt-parsing.html
func (p *Parser) term(maxPriority Integer) (Term, error) {
	var lhs Term
	switch op, err := p.prefix(maxPriority); {
	case err == nil:
		_, rbp := op.bindingPriorities()
		t, err := p.term(rbp)
		if err != nil {
			lhs = op.name
			break
		}
		lhs = op.name.Apply(t)
	default:
		lhs, err = p.term0()
		if err != nil {
			return nil, err
		}
	}

	for {
		op, err := p.infix(maxPriority)
		if err != nil {
			break
		}
		switch _, rbp := op.bindingPriorities(); {
		case rbp > 1200:
			lhs = op.name.Apply(lhs)
		default:
			rhs, err := p.term(rbp)
			if err != nil {
				return nil, err
			}
			lhs = op.name.Apply(lhs, rhs)
		}
	}

	return lhs, nil
}

func (p *Parser) prefix(maxPriority Integer) (operator, error) {
	if p.operators == nil {
		return operator{}, errNoOp
	}

	a, err := p.op(maxPriority)
	if err != nil {
		return operator{}, errNoOp
	}

	if a == "-" {
		switch t := p.next(); t.Kind {
		case TokenInteger, TokenFloatNumber:
			p.backup()
			p.backup()
			return operator{}, errNoOp
		default:
			p.backup()
		}
	}

	switch t := p.next(); t.Kind {
	case TokenOpenCT:
		p.backup()
		p.backup()
		return operator{}, errNoOp
	default:
		p.backup()
	}

	for _, op := range *p.operators {
		if op.name != a {
			continue
		}
		if op.priority > maxPriority {
			continue
		}
		return op, nil
	}

	p.backup()
	return operator{}, errNoOp
}

func (p *Parser) infix(maxPriority Integer) (operator, error) {
	if p.operators == nil {
		return operator{}, errNoOp
	}

	a, err := p.op(maxPriority)
	if err != nil {
		return operator{}, errNoOp
	}

	for _, op := range *p.operators {
		if op.name != a {
			continue
		}
		l, _ := op.bindingPriorities()
		if l > maxPriority {
			continue
		}
		return op, nil
	}

	p.backup()
	return operator{}, errNoOp
}

func (p *Parser) op(maxPriority Integer) (Atom, error) {
	if a, err := p.atom(); err == nil {
		switch a {
		case "[]":
			p.backup()
			if p.current().Kind == TokenCloseList {
				p.backup()
			}
			return "", errNoOp
		case "{}":
			p.backup()
			if p.current().Kind == TokenCloseCurly {
				p.backup()
			}
			return "", errNoOp
		default:
			return a, nil
		}
	}

	switch t := p.next(); t.Kind {
	case TokenComma:
		if maxPriority >= 1000 {
			return Atom(t.Val), nil
		}
	case TokenBar:
		return Atom(t.Val), nil
	}

	p.backup()
	return "", errExpectation
}

func (p *Parser) term0() (Term, error) {
	switch t := p.next(); t.Kind {
	case TokenOpen, TokenOpenCT:
		t, err := p.term(1201)
		if err != nil {
			return nil, err
		}
		if t := p.next(); t.Kind != TokenClose {
			p.backup()
			return nil, errExpectation
		}
		return t, nil
	case TokenInteger:
		return parseInteger(1, t.Val)
	case TokenFloatNumber:
		return parseFloat(1, t.Val)
	case TokenVariable:
		if t.Val == "_" {
			return NewVariable(), nil
		}
		if p.vars == nil {
			return Variable(t.Val), nil
		}
		n := Atom(t.Val)
		for i, v := range *p.vars {
			if v.Name == n {
				(*p.vars)[i].Count++
				return v.Variable, nil
			}
		}
		w := NewVariable()
		*p.vars = append(*p.vars, ParsedVariable{Name: n, Variable: w, Count: 1})
		return w, nil
	case TokenOpenList:
		if t := p.next(); t.Kind == TokenCloseList {
			p.backup()
			p.backup()
			break
		}
		p.backup()
		return p.list()
	case TokenOpenCurly:
		if t := p.next(); t.Kind == TokenCloseCurly {
			p.backup()
			p.backup()
			break
		}
		p.backup()
		return p.curlyBracketedTerm()
	case TokenDoubleQuotedList:
		switch p.doubleQuotes {
		case doubleQuotesChars:
			var chars []Term
			for _, r := range unDoubleQuote(t.Val) {
				chars = append(chars, Atom(r))
			}
			return List(chars...), nil
		case doubleQuotesCodes:
			var codes []Term
			for _, r := range unDoubleQuote(t.Val) {
				codes = append(codes, Integer(r))
			}
			return List(codes...), nil
		default:
			p.backup()
			break
		}
	default:
		p.backup()
	}

	a, err := p.atom()
	if err != nil {
		return nil, err
	}

	if a == "-" {
		switch t := p.next(); t.Kind {
		case TokenInteger:
			return parseInteger(-1, t.Val)
		case TokenFloatNumber:
			return parseFloat(-1, t.Val)
		default:
			p.backup()
		}
	}

	t, err := p.functionalNotation(a)
	if err != nil {
		return nil, err
	}

	if p.placeholder != "" && t == p.placeholder {
		if len(p.args) == 0 {
			return nil, errPlaceholder
		}
		t, p.args = p.args[0], p.args[1:]
	}

	return t, nil
}

func (p *Parser) atom() (Atom, error) {
	if a, err := p.name(); err == nil {
		return a, nil
	}

	switch t := p.next(); t.Kind {
	case TokenOpenList:
		switch t := p.next(); t.Kind {
		case TokenCloseList:
			return "[]", nil
		default:
			p.backup()
			p.backup()
			return "", errExpectation
		}
	case TokenOpenCurly:
		switch t := p.next(); t.Kind {
		case TokenCloseCurly:
			return "{}", nil
		default:
			p.backup()
			p.backup()
			return "", errExpectation
		}
	case TokenDoubleQuotedList:
		switch p.doubleQuotes {
		case doubleQuotesAtom:
			return Atom(unDoubleQuote(t.Val)), nil
		default:
			p.backup()
			return "", errExpectation
		}
	default:
		p.backup()
		return "", errExpectation
	}
}

func (p *Parser) name() (Atom, error) {
	switch t := p.next(); t.Kind {
	case TokenLetterDigit, TokenGraphic, TokenSemicolon, TokenCut:
		return Atom(t.Val), nil
	case TokenQuoted:
		return Atom(unquote(t.Val)), nil
	default:
		p.backup()
		return "", errExpectation
	}
}

func (p *Parser) list() (Term, error) {
	arg, err := p.term(999)
	if err != nil {
		return nil, err
	}
	args := []Term{arg}
	for {
		switch t := p.next(); t.Kind {
		case TokenComma:
			arg, err := p.term(999)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		case TokenBar:
			rest, err := p.term(999)
			if err != nil {
				return nil, err
			}

			switch t := p.next(); t.Kind {
			case TokenCloseList:
				return ListRest(rest, args...), nil
			default:
				p.backup()
				return nil, errExpectation
			}
		case TokenCloseList:
			return List(args...), nil
		default:
			p.backup()
			return nil, errExpectation
		}
	}
}

func (p *Parser) curlyBracketedTerm() (Term, error) {
	t, err := p.term(1201)
	if err != nil {
		return nil, err
	}

	if t := p.next(); t.Kind != TokenCloseCurly {
		p.backup()
		return nil, errExpectation
	}

	return &Compound{
		Functor: "{}",
		Args:    []Term{t},
	}, nil
}

func (p *Parser) functionalNotation(functor Atom) (Term, error) {
	switch t := p.next(); t.Kind {
	case TokenOpenCT:
		arg, err := p.term(999)
		if err != nil {
			return nil, err
		}
		args := []Term{arg}
		for {
			switch t := p.next(); t.Kind {
			case TokenComma:
				arg, err := p.term(999)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			case TokenClose:
				return &Compound{
					Functor: functor,
					Args:    args,
				}, nil
			default:
				p.backup()
				return nil, errExpectation
			}
		}
	default:
		p.backup()
		return functor, nil
	}
}

func parseInteger(sign int64, s string) (Integer, error) {
	base := 10
	switch {
	case strings.HasPrefix(s, "0'"):
		s = s[2:]
		s = quotedIdentEscapePattern.ReplaceAllStringFunc(s, quotedIdentUnescape)
		return Integer(sign * int64([]rune(s)[0])), nil
	case strings.HasPrefix(s, "0b"):
		base = 2
		s = s[2:]
	case strings.HasPrefix(s, "0o"):
		base = 8
		s = s[2:]
	case strings.HasPrefix(s, "0x"):
		base = 16
		s = s[2:]
	}

	f, _, _ := big.ParseFloat(s, base, 0, big.ToZero)
	f.Mul(big.NewFloat(float64(sign)), f)

	switch i, a := f.Int64(); a {
	case big.Above:
		return 0, RepresentationError(FlagMinInteger, nil)
	case big.Below:
		return 0, RepresentationError(FlagMaxInteger, nil)
	default:
		return Integer(i), nil
	}
}

func parseFloat(sign float64, s string) (Float, error) {
	bf, _, _ := big.ParseFloat(s, 10, 0, big.ToZero)
	bf.Mul(big.NewFloat(sign), bf)

	f, _ := bf.Float64()
	return Float(f), nil
}

var (
	quotedIdentEscapePattern  = regexp.MustCompile("''|\\\\(?:[\\nabfnrtv\\\\'\"`]|(?:x[\\da-fA-F]+|[0-8]+)\\\\)")
	doubleQuotedEscapePattern = regexp.MustCompile("\"\"|\\\\(?:[\\nabfnrtv\\\\'\"`]|(?:x[\\da-fA-F]+|[0-8]+)\\\\)")
)

func unquote(s string) string {
	return quotedIdentEscapePattern.ReplaceAllStringFunc(s[1:len(s)-1], quotedIdentUnescape)
}

func quotedIdentUnescape(s string) string {
	switch s {
	case "''":
		return "'"
	case "\\\n":
		return ""
	case `\a`:
		return "\a"
	case `\b`:
		return "\b"
	case `\f`:
		return "\f"
	case `\n`:
		return "\n"
	case `\r`:
		return "\r"
	case `\t`:
		return "\t"
	case `\v`:
		return "\v"
	case `\\`:
		return `\`
	case `\'`:
		return `'`
	case `\"`:
		return `"`
	case "\\`":
		return "`"
	default: // `\x23\` or `\23\`
		s = s[1 : len(s)-1] // `x23` or `23`
		base := 8

		if s[0] == 'x' {
			s = s[1:]
			base = 16
		}

		r, _ := strconv.ParseInt(s, base, 4*8) // rune is up to 4 bytes
		return string(rune(r))
	}
}

func unDoubleQuote(s string) string {
	return doubleQuotedEscapePattern.ReplaceAllStringFunc(s[1:len(s)-1], doubleQuotedUnescape)
}

func doubleQuotedUnescape(s string) string {
	switch s {
	case `""`:
		return `"`
	case "\\\n":
		return ""
	case `\a`:
		return "\a"
	case `\b`:
		return "\b"
	case `\f`:
		return "\f"
	case `\n`:
		return "\n"
	case `\r`:
		return "\r"
	case `\t`:
		return "\t"
	case `\v`:
		return "\v"
	case `\\`:
		return `\`
	case `\'`:
		return `'`
	case `\"`:
		return `"`
	case "\\`":
		return "`"
	default: // `\x23\` or `\23\`
		s = s[1 : len(s)-1] // `x23` or `23`
		base := 8

		if s[0] == 'x' {
			s = s[1:]
			base = 16
		}

		r, _ := strconv.ParseInt(s, base, 4*8) // rune is up to 4 bytes
		return string(rune(r))
	}
}

type tokenRingBuffer struct {
	buf        [4]Token
	start, end int
}

func (b *tokenRingBuffer) put(t Token) {
	b.buf[b.end] = t
	b.end++
	b.end %= len(b.buf)
}

func (b *tokenRingBuffer) get() Token {
	t := b.buf[b.start]
	b.start++
	b.start %= len(b.buf)
	return t
}

func (b *tokenRingBuffer) current() Token {
	return b.buf[b.start]
}

func (b *tokenRingBuffer) empty() bool {
	return b.start == b.end
}

func (b *tokenRingBuffer) backup() {
	b.start--
	b.start %= len(b.buf)
	if b.start < 0 {
		b.start += len(b.buf)
	}
}

type unexpectedTokenError struct {
	actual Token
}

func (e unexpectedTokenError) Error() string {
	return fmt.Sprintf("unexpected token: %s", e.actual)
}
