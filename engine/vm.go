package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ichiban/prolog/nondet"

	"github.com/sirupsen/logrus"
)

const (
	opVoid byte = iota
	opEnter
	opCall
	opExit
	opConst
	opVar
	opFunctor
	opPop

	opCut
)

// VM is the core of a Prolog interpreter. The zero value for VM is a valid VM without any builtin predicates.
type VM struct {
	// OnHalt is a hook which gets triggered right before halt/0 or halt/1.
	OnHalt func()

	// OnArrive is a hook which gets triggered when the execution reached to a procedure.
	OnArrive                       func(goal Term)
	OnExec                         func(op string, arg Term)
	OnCall, OnExit, OnFail, OnRedo func(pi string, args Term)
	OnPanic                        func(r interface{})

	operators       Operators
	procedures      map[procedureIndicator]procedure
	streams         map[Term]*Stream
	input, output   *Stream
	charConversions map[rune]rune
	charConvEnabled bool
	debug           bool
	unknown         unknownAction
}

// SetUserInput sets the given reader as a stream with an alias of user_input.
func (vm *VM) SetUserInput(r io.Reader) {
	const userInput = Atom("user_input")

	s := Stream{
		source: r,
		mode:   streamModeRead,
		alias:  userInput,
	}

	if vm.streams == nil {
		vm.streams = map[Term]*Stream{}
	}
	vm.streams[userInput] = &s

	vm.input = &s
}

// SetUserOutput sets the given writer as a stream with an alias of user_output.
func (vm *VM) SetUserOutput(w io.Writer) {
	const userOutput = Atom("user_output")

	s := Stream{
		sink:  w,
		mode:  streamModeWrite,
		alias: userOutput,
	}

	if vm.streams == nil {
		vm.streams = map[Term]*Stream{}
	}
	vm.streams[userOutput] = &s

	vm.output = &s
}

func (vm *VM) DescribeTerm(t Term) string {
	var buf bytes.Buffer
	_ = t.WriteTerm(&buf, WriteTermOptions{
		Quoted:      true,
		Ops:         vm.operators,
		Descriptive: true,
	})
	return buf.String()
}

// Register0 registers a predicate of arity 0.
func (vm *VM) Register0(name string, p func(nondet.Promise) nondet.Promise) {
	if vm.procedures == nil {
		vm.procedures = map[procedureIndicator]procedure{}
	}
	vm.procedures[procedureIndicator{name: Atom(name), arity: 0}] = predicate0(p)
}

// Register1 registers a predicate of arity 1.
func (vm *VM) Register1(name string, p func(Term, nondet.Promise) nondet.Promise) {
	if vm.procedures == nil {
		vm.procedures = map[procedureIndicator]procedure{}
	}
	vm.procedures[procedureIndicator{name: Atom(name), arity: 1}] = predicate1(p)
}

// Register2 registers a predicate of arity 2.
func (vm *VM) Register2(name string, p func(Term, Term, nondet.Promise) nondet.Promise) {
	if vm.procedures == nil {
		vm.procedures = map[procedureIndicator]procedure{}
	}
	vm.procedures[procedureIndicator{name: Atom(name), arity: 2}] = predicate2(p)
}

// Register3 registers a predicate of arity 3.
func (vm *VM) Register3(name string, p func(Term, Term, Term, nondet.Promise) nondet.Promise) {
	if vm.procedures == nil {
		vm.procedures = map[procedureIndicator]procedure{}
	}
	vm.procedures[procedureIndicator{name: Atom(name), arity: 3}] = predicate3(p)
}

// Register4 registers a predicate of arity 4.
func (vm *VM) Register4(name string, p func(Term, Term, Term, Term, nondet.Promise) nondet.Promise) {
	if vm.procedures == nil {
		vm.procedures = map[procedureIndicator]procedure{}
	}
	vm.procedures[procedureIndicator{name: Atom(name), arity: 4}] = predicate4(p)
}

// Register5 registers a predicate of arity 5.
func (vm *VM) Register5(name string, p func(Term, Term, Term, Term, Term, nondet.Promise) nondet.Promise) {
	if vm.procedures == nil {
		vm.procedures = map[procedureIndicator]procedure{}
	}
	vm.procedures[procedureIndicator{name: Atom(name), arity: 5}] = predicate5(p)
}

type unknownAction int

const (
	unknownError unknownAction = iota
	unknownFail
	unknownWarning
)

func (u unknownAction) String() string {
	switch u {
	case unknownError:
		return "error"
	case unknownFail:
		return "fail"
	case unknownWarning:
		return "warning"
	default:
		return fmt.Sprintf("unknown(%d)", u)
	}
}

type procedure interface {
	Call(*VM, Term, nondet.Promise) nondet.Promise
}

func (vm *VM) arrive(pi procedureIndicator, args Term, k nondet.Promise) nondet.Promise {
	if vm.OnArrive == nil {
		vm.OnArrive = func(goal Term) {}
	}
	var as []Term
	if err := Each(args, func(elem Term) error {
		as = append(as, elem)
		return nil
	}); err != nil {
		return nondet.Error(systemError(err))
	}
	if len(as) == 0 {
		vm.OnArrive(pi.name)
	} else {
		vm.OnArrive(&Compound{Functor: pi.name, Args: as})
	}

	p := vm.procedures[pi]
	if p == nil {
		switch vm.unknown {
		case unknownError:
			return nondet.Error(existenceErrorProcedure(&Compound{
				Functor: "/",
				Args:    []Term{pi.name, pi.arity},
			}))
		case unknownWarning:
			logrus.WithField("procedure", pi).Warn("unknown procedure")
			fallthrough
		case unknownFail:
			return nondet.Bool(false)
		default:
			return nondet.Error(systemError(fmt.Errorf("unknown unknown: %s", vm.unknown)))
		}
	}

	return nondet.Delay(func() nondet.Promise {
		return p.Call(vm, args, k)
	})
}

type cont struct {
	exit nondet.Promise
	fail nondet.Promise
}

func (vm *VM) exec(pc bytecode, xr []Term, vars []*Variable, k cont, args, astack Term) nondet.Promise {
	if vm.OnExec == nil {
		vm.OnExec = func(op string, arg Term) {}
	}
	for len(pc) != 0 {
		switch pc[0] {
		case opVoid:
			vm.OnExec("void", nil)
			pc = pc[1:]
		case opConst:
			x := xr[pc[1]]
			vm.OnExec("const", x)
			var arest Variable
			cons := Compound{
				Functor: ".",
				Args:    []Term{x, &arest},
			}
			if !args.Unify(&cons, false) {
				return k.fail
			}
			pc = pc[2:]
			args = &arest
		case opVar:
			v := vars[pc[1]]
			vm.OnExec("var", v)
			var arest Variable
			cons := Compound{
				Functor: ".",
				Args:    []Term{v, &arest},
			}
			if !args.Unify(&cons, false) {
				return k.fail
			}
			pc = pc[2:]
			args = &arest
		case opFunctor:
			x := xr[pc[1]]
			vm.OnExec("functor", x)
			var arg, arest Variable
			cons1 := Compound{
				Functor: ".",
				Args:    []Term{&arg, &arest},
			}
			if !args.Unify(&cons1, false) {
				return k.fail
			}
			pf, ok := x.(procedureIndicator)
			if !ok {
				return nondet.Error(errors.New("not a principal functor"))
			}
			ok, err := Functor(&arg, pf.name, pf.arity, nondet.Bool(true)).Force()
			if err != nil {
				return nondet.Error(err)
			}
			if !ok {
				return k.fail
			}
			pc = pc[2:]
			args = &Variable{}
			cons2 := Compound{
				Functor: ".",
				Args:    []Term{pf.name, args},
			}
			ok, err = Univ(&arg, &cons2, nondet.Bool(true)).Force()
			if err != nil {
				return nondet.Error(err)
			}
			if !ok {
				return k.fail
			}
			astack = Cons(&arest, astack)
		case opPop:
			vm.OnExec("pop", nil)
			if !args.Unify(List(), false) {
				return k.fail
			}
			pc = pc[1:]
			var a, arest Variable
			cons := Compound{
				Functor: ".",
				Args:    []Term{&a, &arest},
			}
			if !astack.Unify(&cons, false) {
				return k.fail
			}
			args = &a
			astack = &arest
		case opEnter:
			vm.OnExec("enter", nil)
			if !args.Unify(List(), false) {
				return k.fail
			}
			if !astack.Unify(List(), false) {
				return k.fail
			}
			pc = pc[1:]
			var v Variable
			args = &v
			astack = &v
		case opCall:
			x := xr[pc[1]]
			vm.OnExec("call", x)
			if !args.Unify(List(), false) {
				return k.fail
			}
			pc = pc[2:]
			pi, ok := x.(procedureIndicator)
			if !ok {
				return nondet.Error(errors.New("not a principal functor"))
			}
			return nondet.Delay(func() nondet.Promise {
				return vm.arrive(pi, astack, nondet.Delay(func() nondet.Promise {
					var v Variable
					return vm.exec(pc, xr, vars, k, &v, &v)
				}))
			})
		case opExit:
			vm.OnExec("exit", nil)
			return k.exit
		case opCut:
			vm.OnExec("cut", nil)
			pc = pc[1:]
			return nondet.Cut(nondet.Delay(func() nondet.Promise {
				return vm.exec(pc, xr, vars, k, args, astack)
			}))
		default:
			return nondet.Error(fmt.Errorf("unknown(%d)", pc[0]))
		}
	}
	return nondet.Error(errors.New("non-exit end of bytecode"))
}

type clauses []clause

func (cs clauses) Call(vm *VM, args Term, k nondet.Promise) nondet.Promise {
	if len(cs) == 0 {
		return nondet.Bool(false)
	}

	if vm.OnCall == nil {
		vm.OnCall = func(pi string, args Term) {}
	}
	if vm.OnExit == nil {
		vm.OnExit = func(pi string, args Term) {}
	}
	if vm.OnFail == nil {
		vm.OnFail = func(pi string, args Term) {}
	}
	if vm.OnRedo == nil {
		vm.OnRedo = func(pi string, args Term) {}
	}

	fvs := FreeVariables(args)
	ks := make([]func() nondet.Promise, len(cs))
	for i := range cs {
		i, c := i, cs[i]
		ks[i] = func() nondet.Promise {
			if i == 0 {
				vm.OnCall(c.pi.String(), args)
			} else {
				ResetVariables(fvs...)
				vm.OnRedo(c.pi.String(), args)
			}
			vars := make([]*Variable, len(c.vars))
			for i := range c.vars {
				vars[i] = &Variable{}
			}
			return nondet.Delay(func() nondet.Promise {
				return vm.exec(c.bytecode, c.xrTable, vars, cont{
					exit: nondet.Delay(func() nondet.Promise {
						vm.OnExit(c.pi.String(), args)
						return k
					}),
					fail: nondet.Delay(func() nondet.Promise {
						vm.OnFail(c.pi.String(), args)
						ResetVariables(fvs...)
						return nondet.Bool(false)
					}),
				}, args, List())
			})
		}
	}
	return nondet.Opaque(nondet.Delay(ks...))
}

type clause struct {
	pi       procedureIndicator
	raw      Term
	xrTable  []Term
	vars     []*Variable
	bytecode bytecode
}

func (c *clause) compile(t Term) error {
	t = Resolve(t)
	c.raw = t
	switch t := t.(type) {
	case Atom:
		return c.compileClause(t, nil)
	case *Compound:
		if t.Functor == ":-" {
			return c.compileClause(t.Args[0], t.Args[1])
		}
		return c.compileClause(t, nil)
	default:
		return typeErrorCallable(t)
	}
}

func (c *clause) compileClause(head Term, body Term) error {
	switch head := head.(type) {
	case Atom:
	case *Compound:
		for _, a := range head.Args {
			if err := c.compileArg(a); err != nil {
				return err
			}
		}
	default:
		return typeErrorCallable(head)
	}
	if body != nil {
		c.bytecode = append(c.bytecode, opEnter)
		for {
			p, ok := body.(*Compound)
			if !ok || p.Functor != "," || len(p.Args) != 2 {
				break
			}
			if err := c.compilePred(p.Args[0]); err != nil {
				return err
			}
			body = p.Args[1]
		}
		if err := c.compilePred(body); err != nil {
			return err
		}
	}
	c.bytecode = append(c.bytecode, opExit)
	return nil
}

func (c *clause) compilePred(p Term) error {
	switch p := p.(type) {
	case Atom:
		if p == "!" {
			c.bytecode = append(c.bytecode, opCut)
			return nil
		}
		c.bytecode = append(c.bytecode, opCall, c.xrOffset(procedureIndicator{name: p, arity: 0}))
		return nil
	case *Compound:
		for _, a := range p.Args {
			if err := c.compileArg(a); err != nil {
				return err
			}
		}
		c.bytecode = append(c.bytecode, opCall, c.xrOffset(procedureIndicator{name: p.Functor, arity: Integer(len(p.Args))}))
		return nil
	default:
		return typeErrorCallable(p)
	}
}

func (c *clause) compileArg(a Term) error {
	switch a := a.(type) {
	case *Variable:
		c.bytecode = append(c.bytecode, opVar, c.varOffset(a))
	case Float, Integer, Atom:
		c.bytecode = append(c.bytecode, opConst, c.xrOffset(a))
	case *Compound:
		c.bytecode = append(c.bytecode, opFunctor, c.xrOffset(procedureIndicator{name: a.Functor, arity: Integer(len(a.Args))}))
		for _, n := range a.Args {
			if err := c.compileArg(n); err != nil {
				return err
			}
		}
		c.bytecode = append(c.bytecode, opPop)
	default:
		return systemError(fmt.Errorf("unknown argument: %s", a))
	}
	return nil
}

func (c *clause) xrOffset(o Term) byte {
	for i, r := range c.xrTable {
		if r.Unify(o, false) {
			return byte(i)
		}
	}
	c.xrTable = append(c.xrTable, o)
	return byte(len(c.xrTable) - 1)
}

func (c *clause) varOffset(o *Variable) byte {
	for i, v := range c.vars {
		if v == o {
			return byte(i)
		}
	}
	c.vars = append(c.vars, o)
	return byte(len(c.vars) - 1)
}

type bytecode []byte

type predicate0 func(nondet.Promise) nondet.Promise

func (p predicate0) Call(e *VM, args Term, k nondet.Promise) nondet.Promise {
	if !args.Unify(List(), false) {
		return nondet.Error(errors.New("wrong number of arguments"))
	}

	return p(nondet.Delay(func() nondet.Promise {
		return e.exec([]byte{opExit}, nil, nil, cont{
			exit: k,
			fail: nondet.Bool(false),
		}, nil, nil)
	}))
}

type predicate1 func(Term, nondet.Promise) nondet.Promise

func (p predicate1) Call(e *VM, args Term, k nondet.Promise) nondet.Promise {
	var v1 Variable
	if !args.Unify(List(&v1), false) {
		return nondet.Error(fmt.Errorf("wrong number of arguments: %s", args))
	}

	return p(&v1, nondet.Delay(func() nondet.Promise {
		return e.exec([]byte{opExit}, nil, nil, cont{
			exit: k,
			fail: nondet.Bool(false),
		}, nil, nil)
	}))
}

type predicate2 func(Term, Term, nondet.Promise) nondet.Promise

func (p predicate2) Call(e *VM, args Term, k nondet.Promise) nondet.Promise {
	var v1, v2 Variable
	if !args.Unify(List(&v1, &v2), false) {
		return nondet.Error(errors.New("wrong number of arguments"))
	}

	return p(&v1, &v2, nondet.Delay(func() nondet.Promise {
		return e.exec([]byte{opExit}, nil, nil, cont{
			exit: k,
			fail: nondet.Bool(false),
		}, nil, nil)
	}))
}

type predicate3 func(Term, Term, Term, nondet.Promise) nondet.Promise

func (p predicate3) Call(e *VM, args Term, k nondet.Promise) nondet.Promise {
	var v1, v2, v3 Variable
	if !args.Unify(List(&v1, &v2, &v3), false) {
		return nondet.Error(errors.New("wrong number of arguments"))
	}

	return p(&v1, &v2, &v3, nondet.Delay(func() nondet.Promise {
		return e.exec([]byte{opExit}, nil, nil, cont{
			exit: k,
			fail: nondet.Bool(false),
		}, nil, nil)
	}))
}

type predicate4 func(Term, Term, Term, Term, nondet.Promise) nondet.Promise

func (p predicate4) Call(e *VM, args Term, k nondet.Promise) nondet.Promise {
	var v1, v2, v3, v4 Variable
	if !args.Unify(List(&v1, &v2, &v3, &v4), false) {
		return nondet.Error(errors.New("wrong number of arguments"))
	}

	return p(&v1, &v2, &v3, &v4, nondet.Delay(func() nondet.Promise {
		return e.exec([]byte{opExit}, nil, nil, cont{
			exit: k,
			fail: nondet.Bool(false),
		}, nil, nil)
	}))
}

type predicate5 func(Term, Term, Term, Term, Term, nondet.Promise) nondet.Promise

func (p predicate5) Call(e *VM, args Term, k nondet.Promise) nondet.Promise {
	var v1, v2, v3, v4, v5 Variable
	if !args.Unify(List(&v1, &v2, &v3, &v4, &v5), false) {
		return nondet.Error(errors.New("wrong number of arguments"))
	}

	return p(&v1, &v2, &v3, &v4, &v5, nondet.Delay(func() nondet.Promise {
		return e.exec([]byte{opExit}, nil, nil, cont{
			exit: k,
			fail: nondet.Bool(false),
		}, nil, nil)
	}))
}

// FreeVariables extracts variables in the given terms.
func FreeVariables(ts ...Term) []*Variable {
	var fvs []*Variable
	for _, t := range ts {
		fvs = appendFreeVariables(fvs, t)
	}
	return fvs
}

func appendFreeVariables(fvs []*Variable, t Term) []*Variable {
	switch t := t.(type) {
	case *Variable:
		if t.Ref != nil {
			return appendFreeVariables(fvs, t.Ref)
		}
		for _, v := range fvs {
			if v == t {
				return fvs
			}
		}
		return append(fvs, t)
	case *Compound:
		for _, arg := range t.Args {
			fvs = appendFreeVariables(fvs, arg)
		}
	}
	return fvs
}

// ResetVariables resets the assignment of the given variables.
func ResetVariables(vs ...*Variable) {
	for _, v := range vs {
		v.Ref = nil
	}
}
