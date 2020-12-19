package prolog

import (
	"errors"
	"fmt"
	"sort"
)

func (e *Engine) Call(t Term, k func() (bool, error)) (bool, error) {
	name, args, err := nameArgs(t)
	if err != nil {
		return false, err
	}
	return e.arrive(name, args, k)
}

var errCut = errors.New("cut")

func Cut(k func() (bool, error)) (bool, error) {
	ok, err := k()
	if err != nil {
		if errors.Is(err, errCut) {
			return ok, err
		}
		return false, err
	}
	return ok, errCut
}

func Unify(t1, t2 Term, k func() (bool, error)) (bool, error) {
	if !t1.Unify(t2) {
		return false, nil
	}
	return k()
}

func TypeVar(t Term, k func() (bool, error)) (bool, error) {
	if t := Resolve(t); t != nil {
		return false, nil
	}
	return k()
}

func TypeAtom(t Term, k func() (bool, error)) (bool, error) {
	if _, ok := Resolve(t).(Atom); !ok {
		return false, nil
	}
	return k()
}

func TypeInteger(t Term, k func() (bool, error)) (bool, error) {
	if _, ok := Resolve(t).(Integer); !ok {
		return false, nil
	}
	return k()
}

func TypeCompound(t Term, k func() (bool, error)) (bool, error) {
	if _, ok := Resolve(t).(*Compound); !ok {
		return false, nil
	}
	return k()
}

func Functor(term, name, arity Term, k func() (bool, error)) (bool, error) {
	var v *Variable
	for v == nil {
		switch t := term.(type) {
		case Atom:
			if !t.Unify(name) || !Integer(0).Unify(arity) {
				return false, nil
			}
			return k()
		case *Variable:
			if t.Ref == nil {
				v = t
				break
			}
			term = t.Ref
		case *Compound:
			if !t.Functor.Unify(name) || !Integer(len(t.Args)).Unify(arity) {
				return false, nil
			}
			return k()
		default:
			return false, nil
		}
	}

	var n *Atom
	for n == nil {
		switch t := name.(type) {
		case Atom:
			n = &t
		case *Variable:
			if t.Ref == nil {
				return false, errors.New("invalid arguments: atom is not instantiated")
			}
			name = t.Ref
		default:
			return false, fmt.Errorf("invalid arguments: name is %T", name)
		}
	}

	var a *Integer
	for a == nil {
		switch t := arity.(type) {
		case Integer:
			a = &t
		case *Variable:
			if t.Ref == nil {
				return false, errors.New("invalid arguments")
			}
			arity = t.Ref
		default:
			return false, errors.New("invalid arguments")
		}
	}

	if *a == 0 {
		if !v.Unify(*a) {
			return false, nil
		}
		return k()
	}

	vars := make([]Term, *a)
	for i := range vars {
		var v Variable
		vars[i] = &v
	}

	if !v.Unify(&Compound{
		Functor: *n,
		Args:    vars,
	}) {
		return false, nil
	}
	return k()
}

func Univ(term, list Term, k func() (bool, error)) (bool, error) {
	var c *Compound
	for c == nil {
		switch t := term.(type) {
		case *Variable:
			if t.Ref == nil {
				var car, cdr Variable
				if !list.Unify(Cons(&car, &cdr)) {
					return false, errors.New("invalid argument")
				}
				var f *Atom
				for f == nil {
					switch t := car.Ref.(type) {
					case Atom:
						f = &t
					case *Variable:
						if t.Ref == nil {
							return false, errors.New("invalid argument")
						}
						car.Ref = t.Ref
					default:
						return false, errors.New("invalid argument")
					}
				}

				list = cdr.Ref

				var args []Term
				for list != Atom("[]") {
					var car, cdr Variable
					if !list.Unify(Cons(&car, &cdr)) {
						return false, errors.New("invalid argument")
					}
					args = append(args, car.Ref)
					list = cdr.Ref
				}

				if !term.Unify(&Compound{
					Functor: *f,
					Args:    args,
				}) {
					return false, nil
				}
				return k()
			}
			term = t.Ref
		case *Compound:
			c = t
		default:
			return false, errors.New("invalid argument")
		}
	}

	l := List()
	for i := len(c.Args) - 1; i >= 0; i-- {
		l = Cons(c.Args[i], l)
	}
	if !list.Unify(Cons(c.Functor, l)) {
		return false, nil
	}
	return k()
}

func (e *Engine) Op(precedence, typ, name Term, k func() (bool, error)) (bool, error) {
	p, ok := Resolve(precedence).(Integer)
	if !ok {
		return false, fmt.Errorf("invalid precedence: %s", precedence)
	}

	t, ok := Resolve(typ).(Atom)
	if !ok {
		return false, fmt.Errorf("invalid type: %s", typ)
	}

	n, ok := Resolve(name).(Atom)
	if !ok {
		return false, fmt.Errorf("invalid name: %s", name)
	}

	// already defined?
	for i, o := range e.operators {
		if o.Type != t || o.Name != n {
			continue
		}

		// remove it first so that we can insert it again in the right position
		copy(e.operators[i:], e.operators[i+1:])
		e.operators[len(e.operators)-1] = operator{}
		e.operators = e.operators[:len(e.operators)-1]

		// or keep it removed.
		if p == 0 {
			return k()
		}
	}

	// insert
	i := sort.Search(len(e.operators), func(i int) bool {
		return e.operators[i].Precedence >= p
	})
	e.operators = append(e.operators, operator{})
	copy(e.operators[i+1:], e.operators[i:])
	e.operators[i] = operator{
		Precedence: p,
		Type:       t,
		Name:       n,
	}

	return k()
}

func (e *Engine) CurrentOp(precedence, typ, name Term, k func() (bool, error)) (bool, error) {
	a := newAssignment(precedence, typ, name)

	for _, op := range e.operators {
		if op.Precedence.Unify(precedence) && op.Type.Unify(typ) && op.Name.Unify(name) {
			ok, err := k()
			if err != nil {
				if errors.Is(err, errCut) {
					return ok, err
				}
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		a.reset()
	}

	return false, nil
}

func (e *Engine) Assertz(t Term, k func() (bool, error)) (bool, error) {
	name, args, err := nameArgs(t)
	if err != nil {
		return false, err
	}

	switch name {
	case ":-/1": // directive
		var d Variable
		args.Unify(Cons(&d, &Variable{}))
		name, args, err := nameArgs(&d)
		if err != nil {
			return false, err
		}
		ok, err := e.arrive(name, args, k)
		if err != nil {
			return false, err
		}
		return ok, nil
	case ":-/2":
		var h Variable
		args.Unify(Cons(&h, &Variable{}))
		name, _, err = nameArgs(&h)
		if err != nil {
			return false, err
		}
	}

	p, ok := e.procedures[name]
	if !ok {
		p = clauses{}
	}

	cs, ok := p.(clauses)
	if !ok {
		return false, errors.New("builtin")
	}
	c := clause{
		name: name,
	}
	if err := c.compile(t); err != nil {
		return false, err
	}

	e.procedures[name] = append(cs, c)
	return k()
}

func nameArgs(t Term) (string, Term, error) {
	switch f := Resolve(t).(type) {
	case Atom:
		return fmt.Sprintf("%s/0", f), List(), nil
	case *Compound:
		return fmt.Sprintf("%s/%d", f.Functor, len(f.Args)), List(f.Args...), nil
	default:
		return "", nil, errors.New("not callable")
	}
}
