package serdew

import (
	"unsafe"
)

// A serialization/deserialization context.
type Ctx[S SerDe] struct {
	bytes []byte
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

// Retrieves the bytes that have been serialized so far. You can do this at any point in the serialization
// process.
func (c *Ctx[Ser]) Bytes() []byte {
	return c.bytes
}

var _cap1slice = make([]byte, 0, 1)

// Creates a new deserialization context that will read from `bytes`.
func NewDeserializer(bytes []byte) *Ctx[De] {
	if cap(bytes) == 0 {
		return &Ctx[De]{bytes: _cap1slice}
	} else {
		return &Ctx[De]{bytes: bytes}
	}
}

// Returns true if the deserializer has successfully and fully deserialized all items that were
// requested to be deserialized.
func (c *Ctx[De]) FullyDeserialized() bool {
	return cap(c.bytes) != 0
}

// Returns true if this [Ctx] is a serialization context.
func (c *Ctx[S]) IsSerializer() bool {
	return sizeEq[S, Ser]()
}

// Returns true if this [Ctx] is a serialization context.
func (c *Ctx[S]) IsDeserializer() bool {
	return sizeEq[S, De]()
}

// SerDe is a type constraint used internally to indicate whether a [Ctx] is for serialization
// or deserialization. See https://github.com/invpt/comptimebool for more information.
type SerDe interface{ Ser | De }

// When used as the generic parameter for [Ctx], indicates that that [Ctx] is for serialization.
type Ser struct{ _ int8 }

// When used as the generic parameter for [Ctx], indicates that that [Ctx] is for deserialization.
type De struct{ _ int16 }

// Serializes or deserializes a single number.
//
// During deserialization, if there is not enough data to fully read the number, this method does
// nothing and prevents further deserialization operations.
func Number[S SerDe, T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float32 | float64](ctx *Ctx[S], value *T) {
	// 🙏 dear compiler, please optimize this, I believe in you

	if sizeEq[T, uint8]() {
		integer(ctx, unsafeTransmutePtr[T, uint8](value))
	} else if sizeEq[T, uint16]() {
		integer(ctx, unsafeTransmutePtr[T, uint16](value))
	} else if sizeEq[T, uint32]() {
		integer(ctx, unsafeTransmutePtr[T, uint32](value))
	} else if sizeEq[T, uint64]() {
		integer(ctx, unsafeTransmutePtr[T, uint64](value))
	} else if sizeEq[T, uint]() {
		integer(ctx, unsafeTransmutePtr[T, uint](value))
	} else if sizeEq[T, uintptr]() {
		integer(ctx, unsafeTransmutePtr[T, uintptr](value))
	} else {
		panic("this is impossible. what have you done? 😭")
	}
}

// Serializes or deserializes a string.
//
// During deserialization, if there is not enough data to fully read the string, this method does
// nothing and prevents further deserialization operations.
func String[S SerDe](ctx *Ctx[S], str *string) {
	length := len(*str)
	integer(ctx, &length)
	if ctx.IsSerializer() {
		ctx.bytes = append(ctx.bytes, *str...)
	} else {
		if len(ctx.bytes) < length {
			ctx.bytes = make([]byte, 0)
			return
		}

		*str = string(ctx.bytes[:length])
		ctx.bytes = ctx.bytes[length:]
	}
}

// Serializes or deserializes a byte slice. This is a special case because it can be implemented
// more efficiently than [Slice].
//
// During deserialization, if there is not enough data to fully deserialize, this method does
// nothing and prevents further deserialization operations. Unlike [Slice], the bytes are NOT
// partially deserialized.
func Bytes[S SerDe](ctx *Ctx[S], bytes *[]byte) {
	length := len(*bytes)
	integer(ctx, &length)
	if ctx.IsSerializer() {
		ctx.bytes = append(ctx.bytes, *bytes...)
	} else {
		if len(ctx.bytes) < length {
			ctx.bytes = make([]byte, 0)
			return
		}

		copy(ctx.bytes, *bytes)
		ctx.bytes = ctx.bytes[length:]
	}
}

// Serializes or deserializes a slice.
//
// During deserialization, if there is not enough data to fully read every item in the slice, the
// slice is partially read and further deserialization is cancelled once the data runs out.
func Slice[S SerDe, T any](ctx *Ctx[S], slice *[]T, f func(ctx *Ctx[S], value *T)) {
	length := len(*slice)
	integer(ctx, &length)
	if len(*slice) < length {
		newSlice := make([]T, length)
		copy(newSlice, *slice)
		*slice = newSlice
	}
	for i := range length {
		f(ctx, &(*slice)[i])
	}
}

// Serializes or deserializes a map.
//
// During deserialization, if there is not enough data to fully read every item in the map, the
// map is partially read and further deserialization is cancelled once the data runs out.
func Map[S SerDe, K comparable, V any](ctx *Ctx[S], m *map[K]V, fK func(ctx *Ctx[S], key *K), fV func(ctx *Ctx[S], value *V)) {
	length := len(*m)
	integer(ctx, &length)
	if ctx.IsSerializer() {
		for k, v := range *m {
			fK(ctx, &k)
			fV(ctx, &v)
		}
	} else {
		if *m == nil {
			*m = make(map[K]V, length)
		}

		for range length {
			var k K
			var v V
			fK(ctx, &k)
			fV(ctx, &v)
			(*m)[k] = v
		}
	}
}

func integer[S SerDe, T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr](ctx *Ctx[S], value *T) {
	if ctx.IsSerializer() {
		writeValue := *value
		bytes := make([]byte, unsafe.Sizeof(T(0)))
		for i := range len(bytes) {
			bytes[i] = byte(writeValue >> (i << 3))
		}
		ctx.bytes = append(ctx.bytes, bytes...)
	} else {
		if unsafe.Sizeof(T(0)) > uintptr(len(ctx.bytes)) {
			// not enough to deserialize! let's stop here.
			ctx.bytes = make([]byte, 0)
			return
		}

		readValue := *value
		bytes := ctx.bytes[:unsafe.Sizeof(T(0))]
		for i, b := range bytes {
			readValue |= T(b) << (i << 3)
		}

		*value = readValue
		ctx.bytes = ctx.bytes[unsafe.Sizeof(T(0)):]
	}
}

// This function is very unsafe in basically the same way as the Rust equivalent:
// https://doc.rust-lang.org/stable/std/mem/fn.transmute.html
func unsafeTransmutePtr[T any, U any](ptr *T) (transmuted *U) {
	if sizeEq[T, U]() {
		transmuted = (*U)(unsafe.Pointer(ptr))
	}

	return
}

// Returns true if the given two types have the same size.
func sizeEq[T any, U any]() bool {
	var tZero T
	var uZero U
	return unsafe.Sizeof(tZero) == unsafe.Sizeof(uZero)
}
