// Package tuple provides a layer for encoding and decoding multi-element tuples
// into keys usable by FoundationDB. The encoded key maintains the same sort
// order as the original tuple: sorted first by the first element, then by the
// second element, etc. This makes the tuple layer ideal for building a variety
// of higher-level data models.
//
// For general guidance on tuple usage, see the Tuple section of Data Modeling
// (https://foundationdb.com/documentation/data-modeling.html#data-modeling-tuples).
//
// FoundationDB tuples can currently encode byte and unicode strings, integers
// and NULL values. In Go these are represented as []byte, string, int64 and
// nil.
package tuple

import "github.com/abdullin/lex-go"
import "encoding/binary"
import "bytes"
import "fmt"

// A Element is one of the types that may be encoded in FoundationDB
// tuples. Although the Go compiler cannot enforce this, it is a programming
// error to use an unsupported types as a Element (and will typically
// result in a runtime panic).
//
// The valid types for Element are []byte (or lex.KeyConvertible), string,
// int64 (or int), and nil.
type Element interface{}

// Tuple is a slice of objects that can be encoded as FoundationDB tuples. If
// any of the Elements are of unsupported types, a runtime panic will occur
// when the Tuple is packed.
//
// Given a Tuple T containing objects only of these types, then T will be
// identical to the Tuple returned by unpacking the byte slice obtained by
// packing T (modulo type normalization to []byte and int64).
type Tuple []Element

var sizeLimits = []uint64{
	1<<(0*8) - 1,
	1<<(1*8) - 1,
	1<<(2*8) - 1,
	1<<(3*8) - 1,
	1<<(4*8) - 1,
	1<<(5*8) - 1,
	1<<(6*8) - 1,
	1<<(7*8) - 1,
	1<<(8*8) - 1,
}

func encodeBytes(buf *bytes.Buffer, code byte, b []byte) {
	buf.WriteByte(code)
	buf.Write(bytes.Replace(b, []byte{0x00}, []byte{0x00, 0xFF}, -1))
	buf.WriteByte(0x00)
}

func bisectLeft(u uint64) int {
	var n int
	for sizeLimits[n] < u {
		n += 1
	}
	return n
}

func encodeInt(buf *bytes.Buffer, i int64) {
	if i == 0 {
		buf.WriteByte(0x14)
		return
	}

	var n int
	var ibuf bytes.Buffer

	switch {
	case i > 0:
		n = bisectLeft(uint64(i))
		buf.WriteByte(byte(0x14 + n))
		binary.Write(&ibuf, binary.BigEndian, i)
	case i < 0:
		n = bisectLeft(uint64(-i))
		buf.WriteByte(byte(0x14 - n))
		binary.Write(&ibuf, binary.BigEndian, int64(sizeLimits[n])+i)
	}

	buf.Write(ibuf.Bytes()[8-n:])
}

// Pack returns a new byte slice encoding the provided tuple. Pack will panic if
// the tuple contains an element of any type other than []byte,
// lex.KeyConvertible, string, int64, int or nil.
//
// Tuple satisfies the lex.KeyConvertible interface, so it is not necessary to
// call Pack when using a Tuple with a FoundationDB API function that requires a
// key.
func (t Tuple) Pack() []byte {
	buf := new(bytes.Buffer)

	for i, e := range t {
		switch e := e.(type) {
		case nil:
			buf.WriteByte(0x00)
		case int64:
			encodeInt(buf, e)
		case uint32:
			encodeInt(buf, int64(e))
		case uint64:
			encodeInt(buf, int64(e))
		case int:
			encodeInt(buf, int64(e))
		case byte:
			encodeInt(buf, int64(e))
		case []byte:
			encodeBytes(buf, 0x01, e)
		case lex.KeyConvertible:
			encodeBytes(buf, 0x01, []byte(e.LexKey()))
		case string:
			encodeBytes(buf, 0x02, []byte(e))
		default:
			panic(fmt.Sprintf("unencodable element at index %d (%v, type %T)", i, t[i], t[i]))
		}
	}

	return buf.Bytes()
}

func findTerminator(b []byte) int {
	bp := b
	var length int

	for {
		idx := bytes.IndexByte(bp, 0x00)
		length += idx
		if idx+1 == len(bp) || bp[idx+1] != 0xFF {
			break
		}
		length += 2
		bp = bp[idx+2:]
	}

	return length
}

func decodeBytes(b []byte) ([]byte, int) {
	idx := findTerminator(b[1:])
	return bytes.Replace(b[1:idx+1], []byte{0x00, 0xFF}, []byte{0x00}, -1), idx + 2
}

func decodeString(b []byte) (string, int) {
	bp, idx := decodeBytes(b)
	return string(bp), idx
}

func decodeInt(b []byte) (int64, int) {
	if b[0] == 0x14 {
		return 0, 1
	}

	var neg bool

	n := int(b[0]) - 20
	if n < 0 {
		n = -n
		neg = true
	}

	bp := make([]byte, 8)
	copy(bp[8-n:], b[1:n+1])

	var ret int64

	binary.Read(bytes.NewBuffer(bp), binary.BigEndian, &ret)

	if neg {
		ret -= int64(sizeLimits[n])
	}

	return ret, n + 1
}

// Unpack returns the tuple encoded by the provided byte slice, or an error if
// the key does not correctly encode a FoundationDB tuple.
func Unpack(b []byte) (Tuple, error) {
	var t Tuple

	var i int

	for i < len(b) {
		var el interface{}
		var off int

		switch {
		case b[i] == 0x00:
			el = nil
			off = 1
		case b[i] == 0x01:
			el, off = decodeBytes(b[i:])
		case b[i] == 0x02:
			el, off = decodeString(b[i:])
		case 0x0c <= b[i] && b[i] <= 0x1c:
			el, off = decodeInt(b[i:])
		default:
			return nil, fmt.Errorf("unable to decode tuple element with unknown typecode %02x", b[i])
		}

		t = append(t, el)
		i += off
	}

	return t, nil
}

// LexKey returns the packed representation of a Tuple, and allows Tuple to
// satisfy the lex.KeyConvertible interface. LexKey will panic in the same
// circumstances as Pack.
func (t Tuple) LexKey() lex.Key {
	return t.Pack()
}

// LexRangeKeys allows Tuple to satisfy the lex.ExactRange interface. The range
// represents all keys that encode tuples strictly starting with a Tuple (that
// is, all tuples of greater length than the Tuple of which the Tuple is a
// prefix).
func (t Tuple) LexRangeKeys() (lex.KeyConvertible, lex.KeyConvertible) {
	p := t.Pack()
	return lex.Key(concat(p, 0x00)), lex.Key(concat(p, 0xFF))
}

// LexRangeKeySelectors allows Tuple to satisfy the lex.Range interface. The
// range represents all keys that encode tuples strictly starting with a Tuple
// (that is, all tuples of greater length than the Tuple of which the Tuple is a
// prefix).
func (t Tuple) LexRangeKeySelectors() (lex.Selectable, lex.Selectable) {
	b, e := t.LexRangeKeys()
	return lex.FirstGreaterOrEqual(b), lex.FirstGreaterOrEqual(e)
}

func concat(a []byte, b ...byte) []byte {
	r := make([]byte, len(a)+len(b))
	copy(r, a)
	copy(r[len(a):], b)
	return r
}
