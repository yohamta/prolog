package engine

// ListIterator is an iterator for a list.
type ListIterator struct {
	List         Term
	Env          *Env
	AllowPartial bool

	current Term
	err     error

	// Variables for Brent's cycle detection algorithm
	tortoise, hare Term
	power, lam     int
}

// Next proceeds to the next element of the list and returns true if there's such an element.
func (i *ListIterator) Next() bool {
	if i.hare == nil {
		i.hare = i.Env.Resolve(i.List)
	}
	if i.power == 0 {
		i.power = 1
	}
	if i.lam == 0 {
		i.lam = 1
	}

	if ID(i.tortoise) == ID(i.hare) { // Detected a cycle.
		i.err = TypeError(ValidTypeList, i.List, i.Env)
		return false
	}

	if i.power == i.lam {
		i.tortoise = i.hare
		i.power *= 2
		i.lam = 0
	}

	switch l := i.hare.(type) {
	case Variable:
		if !i.AllowPartial {
			i.err = InstantiationError(i.Env)
		}
		return false
	case Atom:
		if l != atomEmptyList {
			i.err = TypeError(ValidTypeList, i.List, i.Env)
		}
		return false
	case Compound:
		if l.Functor() != atomDot || l.Arity() != 2 {
			i.err = TypeError(ValidTypeList, i.List, i.Env)
			return false
		}

		i.current, i.hare = l.Arg(0), i.Env.Resolve(l.Arg(1))
		i.lam++
		return true
	default:
		i.err = TypeError(ValidTypeList, i.List, i.Env)
		return false
	}
}

// Current returns the current element.
func (i *ListIterator) Current() Term {
	return i.current
}

// Err returns an error.
func (i *ListIterator) Err() error {
	return i.err
}

// Suffix returns the rest of the list.
func (i *ListIterator) Suffix() Term {
	if i.hare == nil {
		return i.List
	}
	return i.hare
}

// SeqIterator is an iterator for a sequence.
type SeqIterator struct {
	Seq Term
	Env *Env

	current Term
}

// Next proceeds to the next element of the sequence and returns true if there's such an element.
func (i *SeqIterator) Next() bool {
	switch s := i.Env.Resolve(i.Seq).(type) {
	case nil:
		return false
	case Compound:
		if s.Functor() != atomComma || s.Arity() != 2 {
			i.current = s
			i.Seq = nil
			return true
		}
		i.Seq = s.Arg(1)
		i.current = s.Arg(0)
		return true
	default:
		i.current = s
		i.Seq = nil
		return true
	}
}

// Current returns the current element.
func (i *SeqIterator) Current() Term {
	return i.current
}

// AltIterator is an iterator for alternatives.
type AltIterator struct {
	Alt Term
	Env *Env

	current Term
}

// Next proceeds to the next element of the alternatives and returns true if there's such an element.
func (i *AltIterator) Next() bool {
	switch a := i.Env.Resolve(i.Alt).(type) {
	case nil:
		return false
	case Compound:
		if a.Functor() != atomSemiColon || a.Arity() != 2 {
			i.current = a
			i.Alt = nil
			return true
		}

		// if-then-else construct
		if c, ok := i.Env.Resolve(a.Arg(0)).(Compound); ok && c.Functor() == atomThen && c.Arity() == 2 {
			i.current = a
			i.Alt = nil
			return true
		}

		i.Alt = a.Arg(1)
		i.current = a.Arg(0)
		return true
	default:
		i.current = a
		i.Alt = nil
		return true
	}
}

// Current returns the current element.
func (i *AltIterator) Current() Term {
	return i.current
}

// AnyIterator is an iterator for a list or a sequence.
type AnyIterator struct {
	Any Term
	Env *Env

	backend interface {
		Next() bool
		Current() Term
	}
}

// Next proceeds to the next element and returns true if there's such an element.
func (i *AnyIterator) Next() bool {
	if i.backend == nil {
		if a, ok := i.Env.Resolve(i.Any).(Compound); ok && a.Functor() == atomDot && a.Arity() == 2 {
			i.backend = &ListIterator{List: i.Any, Env: i.Env}
		} else {
			i.backend = &SeqIterator{Seq: i.Any, Env: i.Env}
		}
	}

	return i.backend.Next()
}

// Current returns the current element.
func (i *AnyIterator) Current() Term {
	return i.backend.Current()
}

// Err returns an error.
func (i *AnyIterator) Err() error {
	b, ok := i.backend.(interface{ Err() error })
	if !ok {
		return nil
	}
	return b.Err()
}
