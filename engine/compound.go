package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Compound is a prolog compound.
type Compound struct {
	Functor Atom
	Args    []Term
}

func (c *Compound) String() string {
	var sb strings.Builder
	_ = Write(&sb, c, defaultWriteTermOptions, nil)
	return sb.String()
}

// Unify unifies the compound with t.
func (c *Compound) Unify(t Term, occursCheck bool, env *Env) (*Env, bool) {
	switch t := env.Resolve(t).(type) {
	case *Compound:
		if c.Functor != t.Functor {
			return env, false
		}
		if len(c.Args) != len(t.Args) {
			return env, false
		}
		var ok bool
		for i := range c.Args {
			env, ok = c.Args[i].Unify(t.Args[i], occursCheck, env)
			if !ok {
				return env, false
			}
		}
		return env, true
	case Variable:
		return t.Unify(c, occursCheck, env)
	default:
		return env, false
	}
}

// Unparse emits tokens that represent the compound.
func (c *Compound) Unparse(emit func(Token), opts WriteTermOptions, env *Env) {
	if c.Functor == "." && len(c.Args) == 2 { // list
		emit(Token{Kind: TokenBracketL, Val: "["})
		env.Resolve(c.Args[0]).Unparse(emit, opts, env)
		t := env.Resolve(c.Args[1])
		for {
			if l, ok := t.(*Compound); ok && l.Functor == "." && len(l.Args) == 2 {
				emit(Token{Kind: TokenComma, Val: ","})
				env.Resolve(l.Args[0]).Unparse(emit, opts, env)
				t = env.Resolve(l.Args[1])
				continue
			}
			if a, ok := t.(Atom); ok && a == "[]" {
				break
			}
			emit(Token{Kind: TokenBar, Val: "|"})
			t.Unparse(emit, opts, env)
			break
		}
		emit(Token{Kind: TokenBracketR, Val: "]"})
		return
	}

	if c.Functor == "{}" && len(c.Args) == 1 { // block
		emit(Token{Kind: TokenBraceL, Val: "{"})
		env.Resolve(c.Args[0]).Unparse(emit, opts, env)
		emit(Token{Kind: TokenBraceR, Val: "}"})
		return
	}

	switch len(c.Args) {
	case 1:
		for _, op := range opts.Ops {
			if op.Name != c.Functor {
				continue
			}
			switch op.Specifier {
			case OperatorSpecifierFX, OperatorSpecifierFY:
				if int(op.Priority) > opts.Priority {
					emit(Token{Kind: TokenParenL, Val: "("})
					defer emit(Token{Kind: TokenParenR, Val: ")"})
				}
				c.Functor.Unparse(emit, opts, env)
				{
					opts := opts
					opts.Priority = int(op.Priority)
					env.Resolve(c.Args[0]).Unparse(emit, opts, env)
				}
				return
			case OperatorSpecifierXF, OperatorSpecifierYF:
				if int(op.Priority) > opts.Priority {
					emit(Token{Kind: TokenParenL, Val: "("})
					defer emit(Token{Kind: TokenParenR, Val: ")"})
				}
				{
					opts := opts
					opts.Priority = int(op.Priority)
					env.Resolve(c.Args[0]).Unparse(emit, opts, env)
				}
				c.Functor.Unparse(emit, opts, env)
				return
			}
		}
	case 2:
		for _, op := range opts.Ops {
			if op.Name != c.Functor {
				continue
			}
			switch op.Specifier {
			case OperatorSpecifierXFX, OperatorSpecifierXFY, OperatorSpecifierYFX:
				if int(op.Priority) > opts.Priority {
					emit(Token{Kind: TokenParenL, Val: "("})
					defer emit(Token{Kind: TokenParenR, Val: ")"})
				}
				{
					opts := opts
					opts.Priority = int(op.Priority)
					env.Resolve(c.Args[0]).Unparse(emit, opts, env)
				}
				c.Functor.Unparse(emit, opts, env)
				{
					opts := opts
					opts.Priority = int(op.Priority)
					env.Resolve(c.Args[1]).Unparse(emit, opts, env)
				}
				return
			}
		}
	}

	if opts.NumberVars && c.Functor == "$VAR" && len(c.Args) == 1 {
		switch n := env.Resolve(c.Args[0]).(type) {
		case Integer:
			const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
			i, j := int(n)%len(letters), int(n)/len(letters)
			if j == 0 {
				s := string(letters[i])
				emit(Token{Kind: TokenVariable, Val: s})
				return
			}
			s := fmt.Sprintf("%s%d", string(letters[i]), j)
			emit(Token{Kind: TokenVariable, Val: s})
			return
		}
	}

	c.Functor.Unparse(emit, opts, env)
	emit(Token{Kind: TokenParenL, Val: "("})
	env.Resolve(c.Args[0]).Unparse(emit, opts, env)
	for _, arg := range c.Args[1:] {
		emit(Token{Kind: TokenComma, Val: ","})
		env.Resolve(arg).Unparse(emit, opts, env)
	}
	emit(Token{Kind: TokenParenR, Val: ")"})
}

// Cons returns a list consists of a first element car and the rest cdr.
func Cons(car, cdr Term) Term {
	return &Compound{
		Functor: ".",
		Args:    []Term{car, cdr},
	}
}

// List returns a list of ts.
func List(ts ...Term) Term {
	return ListRest(Atom("[]"), ts...)
}

// ListRest returns a list of ts followed by rest.
func ListRest(rest Term, ts ...Term) Term {
	l := rest
	for i := len(ts) - 1; i >= 0; i-- {
		l = Cons(ts[i], l)
	}
	return l
}

// Set returns a list of ts which elements are unique.
func Set(ts ...Term) Term {
	if len(ts) < 2 {
		return List(ts...)
	}
	us := make([]Term, len(ts))
	copy(us, ts)
	sort.Slice(us, func(i, j int) bool {
		return compare(us[i], us[j], nil) < 0
	})
	n := 1
	for _, u := range us[1:] {
		if compare(us[n-1], u, nil) == 0 {
			continue
		}
		us[n] = u
		n++
	}
	for i := range us[n:] {
		us[n+i] = nil
	}
	return List(us[:n]...)
}

// EachList iterates over list.
func EachList(list Term, f func(elem Term) error, env *Env) error {
	whole := list
	for {
		switch l := env.Resolve(list).(type) {
		case Variable:
			return InstantiationError(whole)
		case Atom:
			if l != "[]" {
				return typeErrorList(l)
			}
			return nil
		case *Compound:
			if l.Functor != "." || len(l.Args) != 2 {
				return typeErrorList(l)
			}
			if err := f(l.Args[0]); err != nil {
				return err
			}
			list = l.Args[1]
		default:
			return typeErrorList(l)
		}
	}
}

func Slice(list Term, env *Env) (ret []Term, err error) {
	err = EachList(list, func(elem Term) error {
		ret = append(ret, env.Resolve(elem))
		return nil
	}, env)
	return
}

// Seq returns a sequence of ts separated by seq.
func Seq(sep Atom, ts ...Term) Term {
	s, ts := ts[len(ts)-1], ts[:len(ts)-1]
	for i := len(ts) - 1; i >= 0; i-- {
		s = &Compound{
			Functor: sep,
			Args:    []Term{ts[i], s},
		}
	}
	return s
}

// EachSeq iterates over a sequence seq separated by sep.
func EachSeq(seq Term, sep Atom, f func(elem Term) error, env *Env) error {
	for {
		p, ok := env.Resolve(seq).(*Compound)
		if !ok || p.Functor != sep || len(p.Args) != 2 {
			break
		}
		if err := f(p.Args[0]); err != nil {
			return err
		}
		seq = p.Args[1]
	}
	return f(seq)
}

func Each(any Term, f func(elem Term) error, env *Env) error {
	if c, ok := env.Resolve(any).(*Compound); ok && c.Functor == "." && len(c.Args) == 2 {
		return EachList(any, f, env)
	}
	return EachSeq(any, ",", f, env)
}