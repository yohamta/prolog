package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListIterator_Next(t *testing.T) {
	t.Run("proper list", func(t *testing.T) {
		iter := ListIterator{List: List(Atom("a"), Atom("b"), Atom("c"))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("c"), iter.Current())
		assert.False(t, iter.Next())
		assert.NoError(t, iter.Err())
	})

	t.Run("improper list", func(t *testing.T) {
		t.Run("variable", func(t *testing.T) {
			iter := ListIterator{List: ListRest(Variable("X"), Atom("a"), Atom("b"))}
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("a"), iter.Current())
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("b"), iter.Current())
			assert.False(t, iter.Next())
			assert.Equal(t, ErrInstantiation, iter.Err())
		})

		t.Run("atom", func(t *testing.T) {
			iter := ListIterator{List: ListRest(Atom("foo"), Atom("a"), Atom("b"))}
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("a"), iter.Current())
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("b"), iter.Current())
			assert.False(t, iter.Next())
			assert.Equal(t, TypeErrorList(ListRest(Atom("foo"), Atom("a"), Atom("b"))), iter.Err())
		})

		t.Run("compound", func(t *testing.T) {
			iter := ListIterator{List: ListRest(Atom("f").Apply(Integer(0)), Atom("a"), Atom("b"))}
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("a"), iter.Current())
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("b"), iter.Current())
			assert.False(t, iter.Next())
			assert.Equal(t, TypeErrorList(ListRest(Atom("f").Apply(Integer(0)), Atom("a"), Atom("b"))), iter.Err())
		})

		t.Run("other", func(t *testing.T) {
			iter := ListIterator{List: ListRest(&mockTerm{}, Atom("a"), Atom("b"))}
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("a"), iter.Current())
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("b"), iter.Current())
			assert.False(t, iter.Next())
			assert.Equal(t, TypeErrorList(ListRest(&mockTerm{}, Atom("a"), Atom("b"))), iter.Err())
		})
	})
}

func TestSeqIterator_Next(t *testing.T) {
	t.Run("sequence", func(t *testing.T) {
		iter := SeqIterator{Seq: Seq(",", Atom("a"), Atom("b"), Atom("c"))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("c"), iter.Current())
		assert.False(t, iter.Next())
	})

	t.Run("sequence with a trailing compound", func(t *testing.T) {
		iter := SeqIterator{Seq: Seq(",", Atom("a"), Atom("b"), Atom("f").Apply(Atom("c")))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("f").Apply(Atom("c")), iter.Current())
		assert.False(t, iter.Next())
	})
}

func TestAltIterator_Next(t *testing.T) {
	t.Run("alternatives", func(t *testing.T) {
		iter := AltIterator{Alt: Seq(";", Atom("a"), Atom("b"), Atom("c"))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("c"), iter.Current())
		assert.False(t, iter.Next())
	})

	t.Run("alternatives with a trailing compound", func(t *testing.T) {
		iter := AltIterator{Alt: Seq(";", Atom("a"), Atom("b"), Atom("f").Apply(Atom("c")))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("f").Apply(Atom("c")), iter.Current())
		assert.False(t, iter.Next())
	})

	t.Run("if then else", func(t *testing.T) {
		iter := AltIterator{Alt: Seq(";", Atom("->").Apply(Atom("a"), Atom("b")), Atom("c"))}
		assert.True(t, iter.Next())
		assert.Equal(t, Seq(";", Atom("->").Apply(Atom("a"), Atom("b")), Atom("c")), iter.Current())
		assert.False(t, iter.Next())
	})
}

func TestAnyIterator_Next(t *testing.T) {
	t.Run("proper list", func(t *testing.T) {
		iter := AnyIterator{Any: List(Atom("a"), Atom("b"), Atom("c"))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("c"), iter.Current())
		assert.False(t, iter.Next())
		assert.NoError(t, iter.Err())
	})

	t.Run("improper list", func(t *testing.T) {
		t.Run("variable", func(t *testing.T) {
			iter := AnyIterator{Any: ListRest(Variable("X"), Atom("a"), Atom("b"))}
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("a"), iter.Current())
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("b"), iter.Current())
			assert.False(t, iter.Next())
			assert.Equal(t, ErrInstantiation, iter.Err())
		})

		t.Run("atom", func(t *testing.T) {
			iter := AnyIterator{Any: ListRest(Atom("foo"), Atom("a"), Atom("b"))}
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("a"), iter.Current())
			assert.True(t, iter.Next())
			assert.Equal(t, Atom("b"), iter.Current())
			assert.False(t, iter.Next())
			assert.Equal(t, TypeErrorList(ListRest(Atom("foo"), Atom("a"), Atom("b"))), iter.Err())
		})
	})

	t.Run("sequence", func(t *testing.T) {
		iter := AnyIterator{Any: Seq(",", Atom("a"), Atom("b"), Atom("c"))}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("b"), iter.Current())
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("c"), iter.Current())
		assert.False(t, iter.Next())
		assert.NoError(t, iter.Err())
	})

	t.Run("single", func(t *testing.T) {
		iter := AnyIterator{Any: Atom("a")}
		assert.True(t, iter.Next())
		assert.Equal(t, Atom("a"), iter.Current())
		assert.False(t, iter.Next())
		assert.NoError(t, iter.Err())
	})
}