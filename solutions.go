package prolog

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ichiban/prolog/engine"
)

// Solutions is the result of a query. Everytime the Next method is called, it searches for the next solution.
// By calling the Scan method, you can retrieve the content of the solution.
type Solutions struct {
	env    *engine.Env
	vars   []engine.Variable
	more   chan<- bool
	next   <-chan *engine.Env
	err    error
	closed bool
}

// ErrClosed indicates the Solutions are already closed and unable to perform the operation.
var ErrClosed = errors.New("closed")

// Close closes the Solutions and terminates the search for other solutions.
func (s *Solutions) Close() error {
	if s.closed {
		return ErrClosed
	}
	close(s.more)
	s.closed = true
	return nil
}

// Next prepares the next solution for reading with the Scan method. It returns true if it finds another solution,
// or false if there's no further solutions or if there's an error.
func (s *Solutions) Next() bool {
	if s.closed {
		return false
	}
	s.more <- true
	var ok bool
	s.env, ok = <-s.next
	return ok
}

// Scan copies the variable values of the current solution into the specified struct/map.
func (s *Solutions) Scan(dest interface{}) error {
	o := reflect.ValueOf(dest)
	switch o.Kind() {
	case reflect.Ptr:
		o = o.Elem()
		switch o.Kind() {
		case reflect.Struct:
			t := o.Type()

			fields := map[string]reflect.Value{}
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				name := f.Name
				if alias, ok := f.Tag.Lookup("prolog"); ok {
					name = alias
				}
				fields[name] = o.Field(i)
			}

			for _, v := range s.vars {
				n := v.String()
				f, ok := fields[n]
				if !ok {
					continue
				}

				val, err := convert(v, f.Type(), s.env)
				if err != nil {
					return err
				}
				fields[n].Set(val)
			}
		}
		return nil
	case reflect.Map:
		t := o.Type()
		if t.Key() != reflect.TypeOf("") {
			return errors.New("map key is not string")
		}

		for _, v := range s.vars {
			val, err := convert(s.env.Simplify(v), t.Elem(), s.env)
			if err != nil {
				return err
			}
			o.SetMapIndex(reflect.ValueOf(v.String()), val)
		}
		return nil
	default:
		return fmt.Errorf("invalid kind: %s", o.Kind())
	}
}

var errConversion = errors.New("conversion failed")

func convert(t engine.Term, typ reflect.Type, env *engine.Env) (reflect.Value, error) {
	switch typ {
	case reflect.TypeOf((*interface{})(nil)).Elem(), reflect.TypeOf((*engine.Term)(nil)).Elem():
		return reflect.ValueOf(env.Resolve(t)), nil
	}

	switch typ.Kind() {
	case reflect.Float32, reflect.Float64:
		return convertFloat(t, typ, env)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return convertInteger(t, typ, env)
	case reflect.String:
		return convertAtom(t, env)
	case reflect.Slice:
		return convertList(t, typ, env)
	default:
		return reflect.Value{}, errConversion
	}
}

func convertFloat(t engine.Term, typ reflect.Type, env *engine.Env) (reflect.Value, error) {
	if f, ok := env.Resolve(t).(engine.Float); ok {
		return reflect.ValueOf(f).Convert(typ), nil
	}
	return reflect.Value{}, errConversion
}

func convertInteger(t engine.Term, typ reflect.Type, env *engine.Env) (reflect.Value, error) {
	if i, ok := env.Resolve(t).(engine.Integer); ok {
		return reflect.ValueOf(i).Convert(typ), nil
	}
	return reflect.Value{}, errConversion
}

func convertAtom(t engine.Term, env *engine.Env) (reflect.Value, error) {
	if a, ok := env.Resolve(t).(engine.Atom); ok {
		return reflect.ValueOf(string(a)), nil
	}
	return reflect.Value{}, errConversion
}

func convertList(t engine.Term, typ reflect.Type, env *engine.Env) (reflect.Value, error) {
	r := reflect.MakeSlice(reflect.SliceOf(typ.Elem()), 0, 0)
	iter := engine.ListIterator{List: t, Env: env}
	for iter.Next() {
		e, err := convert(iter.Current(), typ.Elem(), env)
		if err != nil {
			return r, err
		}
		r = reflect.Append(r, e)
	}
	return r, iter.Err()
}

// Err returns the error if exists.
func (s *Solutions) Err() error {
	return s.err
}

// Vars returns variable names.
func (s *Solutions) Vars() []string {
	ns := make([]string, 0, len(s.vars))
	for _, v := range s.vars {
		if v.Anonymous() {
			continue
		}
		ns = append(ns, v.String())
	}
	return ns
}

// Solution is the single result of a query.
type Solution struct {
	sols *Solutions
	err  error
}

// Scan copies the variable values of the solution into the specified struct/map.
func (s *Solution) Scan(dest interface{}) error {
	if err := s.err; err != nil {
		return err
	}
	return s.sols.Scan(dest)
}

// Err returns an error that occurred while querying for the Solution, if any.
func (s *Solution) Err() error {
	return s.err
}

// Vars returns variable names.
func (s *Solution) Vars() []string {
	if s.sols == nil {
		return nil
	}
	return s.sols.Vars()
}
