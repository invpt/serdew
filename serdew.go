package serdew

import (
	"encoding"
	"errors"
	"unsafe"
)

// A serialization/deserialization context.
type Ctx[S SerDe] struct {
	bytes []byte
	err   error
}

// Creates a new serialization context. The capacity of the internal serialization buffer starts
// at zero, which isn't the most efficient; see [NewSerializerWithCapacity] and [NewSerializerWithBacking].
func NewSerializer() *Ctx[Ser] {
	return &Ctx[Ser]{}
}

// Creates a new serialization context with the given initial capacity.
func NewSerializerWithCapacity(cap int) *Ctx[Ser] {
	return &Ctx[Ser]{bytes: make([]byte, 0, cap)}
}

// Creates a new serialization context that uses the given slice as its internal serialization buffer.
// This is the most efficient way to serialize data. To go fast, reuse the same buffer over and over.
func NewSerializerWithBacking(bytes []byte) *Ctx[Ser] {
	return &Ctx[Ser]{bytes: bytes[:0]}
}

// Creates a new deserialization context that will read from `bytes`.
func NewDeserializer(bytes []byte) *Ctx[De] {
	return &Ctx[De]{bytes: bytes}
}

// Retrieves the byte buffer of this (de) serializer. Be sure to check for errors with [Error].
//
// If this is a serializer, the byte buffer holds the bytes that have been serialized so far.
//
// If this is a desearializer, the byte buffer holds the bytes that have yet to be deserialized.
func (ctx *Ctx[S]) Bytes() []byte {
	return ctx.bytes
}

var errUnexpectedEnd error = errors.New("unexpected end of buffer during deserialization")

// Retrieves a buffer of n bytes for (de)serialization. This is meant to be used by very custom
// serializers; if you are just writing a serializer for a regular type, you probably want to use
// one of [Number], [String], [Slice], or [Map].
//
// During serialization, the contents of the returned buffer are undefined, and are intended to be
// written to with serialized data.
//
// During deserialization, the contents of the returned buffer are intended to be read from.
func (ctx *Ctx[S]) Raw(n int) (bytes []byte, err error) {
	if ctx.err != nil {
		return nil, ctx.err
	}

	if ctx.IsSerializer() {
		if n+len(ctx.bytes) > cap(ctx.bytes) {
			// we must allocate a bigger buffer.
			newCtxBytes := make([]byte, len(ctx.bytes), n+len(ctx.bytes))
			copy(newCtxBytes, ctx.bytes)
			ctx.bytes = newCtxBytes
		}

		bytes = ctx.bytes[len(ctx.bytes) : n+len(ctx.bytes)]
		ctx.bytes = ctx.bytes[:n+len(ctx.bytes)]
	} else {
		if n > len(ctx.bytes) {
			return nil, ctx.Abort(errUnexpectedEnd)
		}

		bytes = ctx.bytes[:n]
		ctx.bytes = ctx.bytes[n:]
	}

	return
}

// Returns true if this [Ctx] is a [Serializer].
func (c *Ctx[S]) IsSerializer() bool {
	return sizeEq[S, Ser]()
}

// Returns true if this [Ctx] is a [Deserializer].
func (c *Ctx[S]) IsDeserializer() bool {
	return sizeEq[S, De]()
}

// Returns the error, if any, indicating what has gone wrong during (de)serialization so far.
func (ctx *Ctx[S]) Error() error {
	return ctx.err
}

// Aborts further (de)serialization. err must be non-nil. You should use this function in a return
// statement, like so:
//
//	return ctx.Abort(err)
func (c *Ctx[S]) Abort(err error) error {
	if err == nil {
		panic("serdew.Ctx[S].Abort() called with nil error")
	}

	if c.err == nil {
		c.err = err
	}

	return c.err
}

// SerDe is a type constraint used to indicate whether a [Ctx] is for serialization
// or deserialization. See https://github.com/invpt/comptimebool for more information.
type SerDe interface{ Ser | De }

// When used as the generic parameter for [Ctx], indicates that that [Ctx] is for serialization.
type Ser struct{ _ int8 }

// When used as the generic parameter for [Ctx], indicates that that [Ctx] is for deserialization.
type De struct{ _ int16 }

// Serializes or deserializes a number.
func Number[S SerDe, T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float32 | float64](ctx *Ctx[S], value *T) (err error) {
	if canTransmute[T, uint8]() {
		err = integer(ctx, unsafeTransmutePtr[T, uint8](value))
	} else if canTransmute[T, uint16]() {
		err = integer(ctx, unsafeTransmutePtr[T, uint16](value))
	} else if canTransmute[T, uint32]() {
		err = integer(ctx, unsafeTransmutePtr[T, uint32](value))
	} else if canTransmute[T, uint64]() {
		err = integer(ctx, unsafeTransmutePtr[T, uint64](value))
	} else if canTransmute[T, uint]() {
		err = integer(ctx, unsafeTransmutePtr[T, uint](value))
	} else if canTransmute[T, uintptr]() {
		err = integer(ctx, unsafeTransmutePtr[T, uintptr](value))
	} else {
		// this is extremely unlikely and may literally never happen.
		// it should only happen when serdew's alignment assumptions are incorrect.
		// serdew assumes that
		//   (a) each pair of intX and uintX types has identical alignment. i know of no computer
		//       where this would not be true.
		//   (b) each floatX type has alignment that is a multiple of and greater than or equal to
		//       the alignment of the corresponding intX/uintX type. this situation is more plausible,
		//       but i still don't know of a system where this is the case.
		// in the event that such a system does exist, serdew's alignment assumptions could be reduced
		// at the cost of a greater amount of code in this file (but likely no performance cost)
		panic("your system is incompatible with the serdew library. file a bug?")
	}

	return
}

// Serializes or deserializes a string.
func String[S SerDe](ctx *Ctx[S], str *string) error {
	length := len(*str)
	err := integer(ctx, &length)
	if err != nil {
		return err
	}

	bytes, err := ctx.Raw(length)
	if err != nil {
		return err
	}

	if ctx.IsSerializer() {
		copy(bytes, *str)
	} else {
		*str = string(bytes)
	}

	return nil
}

// Serializes or deserializes a byte slice. This is a special case because it can be implemented
// more efficiently than [Slice].
func Bytes[S SerDe](ctx *Ctx[S], bytes *[]byte) error {
	length := len(*bytes)
	err := integer(ctx, &length)
	if err != nil {
		return err
	}

	ctxBytes, err := ctx.Raw(length)
	if err != nil {
		return err
	}

	if ctx.IsSerializer() {
		copy(ctxBytes, *bytes)
	} else {
		if cap(*bytes) < len(ctxBytes) {
			*bytes = make([]byte, len(ctxBytes))
		}
		copy((*bytes)[:len(ctxBytes)], ctxBytes)
	}

	return nil
}

// Serializes or deserializes a slice.
func Slice[S SerDe, T any](ctx *Ctx[S], slice *[]T, f func(ctx *Ctx[S], value *T) error) error {
	length := len(*slice)
	err := integer(ctx, &length)
	if err != nil {
		return err
	}

	if len(*slice) < length {
		newSlice := make([]T, length)
		copy(newSlice, *slice)
		*slice = newSlice
	}
	for i := range length {
		err := f(ctx, &(*slice)[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// Serializes or deserializes a map.
func Map[S SerDe, K comparable, V any](ctx *Ctx[S], m *map[K]V, fK func(ctx *Ctx[S], key *K) error, fV func(ctx *Ctx[S], value *V) error) error {
	length := len(*m)
	err := integer(ctx, &length)
	if err != nil {
		return err
	}

	if ctx.IsSerializer() {
		for k, v := range *m {
			if err := fK(ctx, &k); err != nil {
				return err
			}
			if err := fV(ctx, &v); err != nil {
				return err
			}
		}
	} else {
		if *m == nil {
			*m = make(map[K]V, length)
		}

		for range length {
			var k K
			var v V
			if err := fK(ctx, &k); err != nil {
				return err
			}
			if err := fV(ctx, &v); err != nil {
				return err
			}
			(*m)[k] = v
		}
	}

	return nil
}

type BinaryMarshalerUnmarshaler interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// Uses the [encoding.BinaryMarshaler] and [encoding.BinaryUnmarshaler] interfaces to (de)serialize
// the given value.
func Binary[S SerDe, T BinaryMarshalerUnmarshaler](ctx *Ctx[S], value T) error {
	if ctx.IsSerializer() {
		data, err := value.MarshalBinary()
		if err != nil {
			return ctx.Abort(err)
		}

		length := len(data)
		integer(ctx, &length)

		bytes, err := ctx.Raw(length)
		if err != nil {
			return err
		}

		copy(bytes, data)
	} else {
		length := 0
		integer(ctx, &length)

		bytes, err := ctx.Raw(length)
		if err != nil {
			return err
		}

		if err := value.UnmarshalBinary(bytes); err != nil {
			return err
		}
	}

	return nil
}

// Serializes or deserializes an integer. The public interface to this function is [Number].
func integer[S SerDe, T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr](ctx *Ctx[S], value *T) error {
	bytes, err := ctx.Raw(int(unsafe.Sizeof(T(0))))
	if err != nil {
		return err
	}

	if ctx.IsSerializer() {
		writeValue := *value
		for i := range len(bytes) {
			bytes[i] = byte(writeValue >> (i << 3))
		}
	} else {
		var readValue T
		for i, b := range bytes {
			readValue |= T(b) << (i << 3)
		}
		*value = readValue
	}

	return nil
}

// Very unsafe in basically the same way as the Rust equivalent:
// https://doc.rust-lang.org/stable/std/mem/fn.transmute.html
func unsafeTransmutePtr[T any, U any](ptr *T) (transmuted *U) {
	if canTransmute[T, U]() {
		transmuted = (*U)(unsafe.Pointer(ptr))
	} else {
		// see the comment in [Number] about why this is unlikely
		panic("your system is incompatible with the serdew library. file a bug?")
	}

	return
}

// Returns true if [unsafeTransmutePtr] would succeed for the given pair of types.
func canTransmute[T any, U any]() bool {
	return sizeEq[T, U]() && alignCompat[T, U]()
}

// Returns true if the given two types have the same size.
func sizeEq[T any, U any]() bool {
	var tZero T
	var uZero U
	return unsafe.Sizeof(tZero) == unsafe.Sizeof(uZero)
}

// Returns true if U's alignment is compatible with T's alignment.
func alignCompat[T any, U any]() bool {
	var tZero T
	var uZero U
	return unsafe.Alignof(tZero) >= unsafe.Alignof(uZero) && unsafe.Alignof(tZero)%unsafe.Alignof(uZero) == 0
}
