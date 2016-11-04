// Package lex contains functions to compose and manipulate
// lexicographically sortable keys. It is forked from FoundationDB API libs (RIP).
package lex

// A Range describes all keys between a begin (inclusive) and end (exclusive)
// key selector.
type Range interface {
	// LexRangeKeySelectors returns a pair of key selectors that describe the
	// beginning and end of a range.
	LexRangeKeySelectors() (begin, end Selectable)
}

// An ExactRange describes all keys between a begin (inclusive) and end
// (exclusive) key. If you need to specify an ExactRange and you have only a
// Range, you must resolve the selectors returned by
// (Range).LexRangeKeySelectors to keys using the (Transaction).GetKey method.
//
// Any object that implements ExactRange also implements Range, and may be used
// accordingly.
type ExactRange interface {
	// LexRangeKeys returns a pair of keys that describe the beginning and end
	// of a range.
	LexRangeKeys() (begin, end KeyConvertible)

	// An object that implements ExactRange must also implement Range
	// (logically, by returning FirstGreaterOrEqual of the keys returned by
	// LexRangeKeys).
	Range
}

// KeyRange is an ExactRange constructed from a pair of KeyConvertibles. Note
// that the default zero-value of KeyRange specifies an empty range before all
// keys in the database.
type KeyRange struct {
	Begin, End KeyConvertible
}

// LexRangeKeys allows KeyRange to satisfy the ExactRange interface.
func (kr KeyRange) LexRangeKeys() (KeyConvertible, KeyConvertible) {
	return kr.Begin, kr.End
}

// LexRangeKeySelectors allows KeyRange to satisfy the Range interface.
func (kr KeyRange) LexRangeKeySelectors() (Selectable, Selectable) {
	return FirstGreaterOrEqual(kr.Begin), FirstGreaterOrEqual(kr.End)
}

// SelectorRange is a Range constructed directly from a pair of Selectable
// objects. Note that the default zero-value of SelectorRange specifies an empty
// range before all keys in the database.
type SelectorRange struct {
	Begin, End Selectable
}

// LexRangeKeySelectors allows SelectorRange to satisfy the Range interface.
func (sr SelectorRange) LexRangeKeySelectors() (Selectable, Selectable) {
	return sr.Begin, sr.End
}
