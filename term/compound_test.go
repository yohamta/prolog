package term

import (
	"testing"

	"github.com/ichiban/prolog/syntax"

	"github.com/stretchr/testify/assert"
)

func TestCompound_Unify(t *testing.T) {
	unit := Compound{
		Functor: "foo",
		Args:    []Interface{Atom("bar")},
	}

	t.Run("atom", func(t *testing.T) {
		env, ok := unit.Unify(Atom("foo"), false, nil)
		assert.False(t, ok)
		assert.Nil(t, env)
	})

	t.Run("integer", func(t *testing.T) {
		env, ok := unit.Unify(Integer(1), false, nil)
		assert.False(t, ok)
		assert.Nil(t, env)
	})

	t.Run("variable", func(t *testing.T) {
		t.Run("free", func(t *testing.T) {
			v := Variable("X")
			env, ok := unit.Unify(v, false, nil)
			assert.True(t, ok)
			assert.Equal(t, &unit, env.Resolve(v))
		})
		t.Run("bound to the same value", func(t *testing.T) {
			v := Variable("X")
			env := NewEnv().
				Bind(v, &unit)
			_, ok := unit.Unify(v, false, env)
			assert.True(t, ok)
		})
		t.Run("bound to a different value", func(t *testing.T) {
			v := Variable("X")
			env := NewEnv().
				Bind(v, &Compound{
					Functor: "foo",
					Args:    []Interface{Atom("baz")},
				})

			_, ok := unit.Unify(&v, false, env)
			assert.False(t, ok)
		})
	})

	t.Run("compound", func(t *testing.T) {
		_, ok := unit.Unify(&Compound{
			Functor: "foo",
			Args:    []Interface{Atom("bar")},
		}, false, nil)
		assert.True(t, ok)
		_, ok = unit.Unify(&Compound{
			Functor: "foo",
			Args:    []Interface{Atom("bar"), Atom("baz")},
		}, false, nil)
		assert.False(t, ok)
		_, ok = unit.Unify(&Compound{
			Functor: "baz",
			Args:    []Interface{Atom("bar")},
		}, false, nil)
		assert.False(t, ok)
		_, ok = unit.Unify(&Compound{
			Functor: "foo",
			Args:    []Interface{Atom("baz")},
		}, false, nil)
		assert.False(t, ok)
		v := Variable("X")
		env, ok := unit.Unify(&Compound{
			Functor: "foo",
			Args:    []Interface{v},
		}, false, nil)
		assert.True(t, ok)
		assert.Equal(t, Atom("bar"), env.Resolve(v))
	})
}

func TestCompound_Unparse(t *testing.T) {
	t.Run("operator precedence", func(t *testing.T) {
		c := Compound{
			Functor: "*",
			Args: []Interface{
				Integer(2),
				&Compound{
					Functor: "+",
					Args: []Interface{
						Integer(2),
						Integer(2),
					},
				},
			},
		}

		ops := Operators{
			{Priority: 500, Specifier: OperatorSpecifierYFX, Name: `+`},
			{Priority: 400, Specifier: OperatorSpecifierYFX, Name: `*`},
		}

		var tokens []syntax.Token
		c.Unparse(func(token syntax.Token) {
			tokens = append(tokens, token)
		}, WriteTermOptions{Ops: ops, Priority: 1200}, nil)
		assert.Equal(t, []syntax.Token{
			{Kind: syntax.TokenInteger, Val: "2"},
			{Kind: syntax.TokenGraphic, Val: "*"},
			{Kind: syntax.TokenParenL, Val: "("},
			{Kind: syntax.TokenInteger, Val: "2"},
			{Kind: syntax.TokenGraphic, Val: "+"},
			{Kind: syntax.TokenInteger, Val: "2"},
			{Kind: syntax.TokenParenR, Val: ")"},
		}, tokens)
	})

	t.Run("ignore_ops", func(t *testing.T) {
		c := Compound{
			Functor: "+",
			Args: []Interface{
				Integer(2),
				Integer(-2),
			},
		}

		t.Run("false", func(t *testing.T) {
			ops := Operators{
				{Priority: 500, Specifier: OperatorSpecifierYFX, Name: "+"},
				{Priority: 200, Specifier: OperatorSpecifierFY, Name: "-"},
			}

			var tokens []syntax.Token
			c.Unparse(func(token syntax.Token) {
				tokens = append(tokens, token)
			}, WriteTermOptions{Ops: ops, Priority: 1200}, nil)
			assert.Equal(t, []syntax.Token{
				{Kind: syntax.TokenInteger, Val: "2"},
				{Kind: syntax.TokenGraphic, Val: "+"},
				{Kind: syntax.TokenSign, Val: "-"},
				{Kind: syntax.TokenInteger, Val: "2"},
			}, tokens)
		})

		t.Run("true", func(t *testing.T) {
			var tokens []syntax.Token
			c.Unparse(func(token syntax.Token) {
				tokens = append(tokens, token)
			}, WriteTermOptions{Ops: nil, Priority: 1200}, nil)
			assert.Equal(t, []syntax.Token{
				{Kind: syntax.TokenGraphic, Val: "+"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenInteger, Val: "2"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenSign, Val: "-"},
				{Kind: syntax.TokenInteger, Val: "2"},
				{Kind: syntax.TokenParenR, Val: ")"},
			}, tokens)
		})
	})

	t.Run("numbervars", func(t *testing.T) {
		c := Compound{
			Functor: "f",
			Args: []Interface{
				&Compound{Functor: "$VAR", Args: []Interface{Integer(0)}},
				&Compound{Functor: "$VAR", Args: []Interface{Integer(1)}},
				&Compound{Functor: "$VAR", Args: []Interface{Integer(25)}},
				&Compound{Functor: "$VAR", Args: []Interface{Integer(26)}},
				&Compound{Functor: "$VAR", Args: []Interface{Integer(27)}},
			},
		}

		t.Run("false", func(t *testing.T) {
			var tokens []syntax.Token
			c.Unparse(func(token syntax.Token) {
				tokens = append(tokens, token)
			}, WriteTermOptions{NumberVars: false, Priority: 1200}, nil)
			assert.Equal(t, []syntax.Token{
				{Kind: syntax.TokenIdent, Val: "f"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenIdent, Val: "$VAR"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenInteger, Val: "0"},
				{Kind: syntax.TokenParenR, Val: ")"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenIdent, Val: "$VAR"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenInteger, Val: "1"},
				{Kind: syntax.TokenParenR, Val: ")"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenIdent, Val: "$VAR"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenInteger, Val: "25"},
				{Kind: syntax.TokenParenR, Val: ")"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenIdent, Val: "$VAR"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenInteger, Val: "26"},
				{Kind: syntax.TokenParenR, Val: ")"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenIdent, Val: "$VAR"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenInteger, Val: "27"},
				{Kind: syntax.TokenParenR, Val: ")"},
				{Kind: syntax.TokenParenR, Val: ")"},
			}, tokens)
		})

		t.Run("true", func(t *testing.T) {
			var tokens []syntax.Token
			c.Unparse(func(token syntax.Token) {
				tokens = append(tokens, token)
			}, WriteTermOptions{NumberVars: true, Priority: 1200}, nil)
			assert.Equal(t, []syntax.Token{
				{Kind: syntax.TokenIdent, Val: "f"},
				{Kind: syntax.TokenParenL, Val: "("},
				{Kind: syntax.TokenVariable, Val: "A"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenVariable, Val: "B"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenVariable, Val: "Z"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenVariable, Val: "A1"},
				{Kind: syntax.TokenComma, Val: ","},
				{Kind: syntax.TokenVariable, Val: "B1"},
				{Kind: syntax.TokenParenR, Val: ")"},
			}, tokens)
		})
	})
}

func TestSet(t *testing.T) {
	assert.Equal(t, List(), Set())
	assert.Equal(t, List(Atom("a")), Set(Atom("a")))
	assert.Equal(t, List(Atom("a")), Set(Atom("a"), Atom("a"), Atom("a")))
	assert.Equal(t, List(Atom("a"), Atom("b"), Atom("c")), Set(Atom("c"), Atom("b"), Atom("a")))
}
