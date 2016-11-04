// Package subspace provides a convenient way to use FoundationDB tuples to
// define namespaces for different categories of data. The namespace is
// specified by a prefix tuple which is prepended to all tuples packed by the
// subspace. When unpacking a key with the subspace, the prefix tuple will be
// removed from the result.
//
// As a best practice, API clients should use at least one subspace for
// application data. For general guidance on subspace usage, see the Subspaces
// section of the Developer Guide
// (https://foundationdb.com/documentation/developer-guide.html#developer-guide-sub-keyspaces).
package subspace

import (
	"bytes"
	"errors"

	"github.com/abdullin/lex-go/tuple"

	"github.com/abdullin/lex-go"
)

// Subspace represents a well-defined region of keyspace in a FoundationDB
// database.
type Subspace interface {
	// Sub returns a new Subspace whose prefix extends this Subspace with the
	// encoding of the provided element(s). If any of the elements are not a
	// valid tuple.Element, Sub will panic.
	Sub(el ...tuple.Element) Subspace

	// Bytes returns the literal bytes of the prefix of this Subspace.
	Bytes() []byte

	// Pack returns the key encoding the specified Tuple with the prefix of this
	// Subspace prepended.
	Pack(t tuple.Tuple) lex.Key

	// Unpack returns the Tuple encoded by the given key with the prefix of this
	// Subspace removed. Unpack will return an error if the key is not in this
	// Subspace or does not encode a well-formed Tuple.
	Unpack(k lex.KeyConvertible) (tuple.Tuple, error)

	// Contains returns true if the provided key starts with the prefix of this
	// Subspace, indicating that the Subspace logically contains the key.
	Contains(k lex.KeyConvertible) bool

	// All Subspaces implement lex.KeyConvertible and may be used as
	// FoundationDB keys (corresponding to the prefix of this Subspace).
	lex.KeyConvertible

	// All Subspaces implement lex.ExactRange and lex.Range, and describe all
	// keys logically in this Subspace.
	lex.ExactRange
}

type subspace struct {
	b []byte
}

// AllKeys returns the Subspace corresponding to all keys in a FoundationDB
// database.
func AllKeys() Subspace {
	return subspace{}
}

// Sub returns a new Subspace whose prefix is the encoding of the provided
// element(s). If any of the elements are not a valid tuple.Element, a
// runtime panic will occur.
func Sub(el ...tuple.Element) Subspace {
	return subspace{tuple.Tuple(el).Pack()}
}

// FromBytes returns a new Subspace from the provided bytes.
func FromBytes(b []byte) Subspace {
	s := make([]byte, len(b))
	copy(s, b)
	return subspace{s}
}

func (s subspace) Sub(el ...tuple.Element) Subspace {
	return subspace{concat(s.Bytes(), tuple.Tuple(el).Pack()...)}
}

func (s subspace) Bytes() []byte {
	return s.b
}

func (s subspace) Pack(t tuple.Tuple) lex.Key {
	return lex.Key(concat(s.b, t.Pack()...))
}

func (s subspace) Unpack(k lex.KeyConvertible) (tuple.Tuple, error) {
	key := k.LexKey()
	if !bytes.HasPrefix(key, s.b) {
		return nil, errors.New("key is not in subspace")
	}
	return tuple.Unpack(key[len(s.b):])
}

func (s subspace) Contains(k lex.KeyConvertible) bool {
	return bytes.HasPrefix(k.LexKey(), s.b)
}

func (s subspace) LexKey() lex.Key {
	return lex.Key(s.b)
}

func (s subspace) LexRangeKeys() (lex.KeyConvertible, lex.KeyConvertible) {
	return lex.Key(concat(s.b, 0x00)), lex.Key(concat(s.b, 0xFF))
}

func (s subspace) LexRangeKeySelectors() (lex.Selectable, lex.Selectable) {
	begin, end := s.LexRangeKeys()
	return lex.FirstGreaterOrEqual(begin), lex.FirstGreaterOrEqual(end)
}

func concat(a []byte, b ...byte) []byte {
	r := make([]byte, len(a)+len(b))
	copy(r, a)
	copy(r[len(a):], b)
	return r
}
