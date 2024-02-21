# serdew
A tiny Go library for binary serialization with a focus on performance. You'll need to write a bit of extra code, but you won't need to write separate Marshal/Unmarshal methods, which are the easiest way to introduce serialization bugs. Serdew completely avoids using reflection.

## Example
Here's an example (available as well in `example/main.go`) showcasing some of the features of serdew:
```go
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/invpt/serdew"
)

type Person struct {
	Name       string
	Age        int
	Height     float64
	Address    []string
	Friends    []Person
	ExampleMap map[string]int
	LastOnline time.Time
}

func SerdewPerson[S serdew.SerDe](ctx *serdew.Ctx[S], p *Person) error {
	serdew.String(ctx, &p.Name)
	serdew.Number(ctx, &p.Age)
	serdew.Number(ctx, &p.Height)
	serdew.Slice(ctx, &p.Address, serdew.String)
	serdew.Slice(ctx, &p.Friends, SerdewPerson)
	serdew.Map(ctx, &p.ExampleMap, serdew.String, serdew.Number)
	serdew.Binary(ctx, &p.LastOnline)
	return ctx.Error()
}

func main() {
	person1 := Person{
		Name:    "James",
		Age:     25,
		Height:  1.7,
		Address: []string{"123 Fake Rd"},
		Friends: []Person{
			{
				Name:       "Mary",
				Age:        26,
				Height:     1.5,
				Address:    []string{"Mary's House"},
				Friends:    []Person{},
				LastOnline: time.Now().Add(-time.Hour * 24),
			},
			{
				Name:    "Bob",
				Age:     45,
				Height:  1.6,
				Address: []string{"628 Apt. Ave", "Apartment 23"},
				Friends: []Person{
					{
						Name:    "Nested Friend",
						Age:     84829340,
						Height:  math.Inf(1),
						Address: []string{},
						Friends: []Person{},
					},
				},
				LastOnline: time.Now().Add(-time.Hour),
			},
		},
		ExampleMap: map[string]int{
			"test1": 1,
			"test2": 29847329,
		},
		LastOnline: time.Now(),
	}

	fmt.Println("Pre-serialized person:", person1)

	ser := serdew.NewSerializer()
	if err := SerdewPerson(ser, &person1); err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Serialized bytes:", ser.Bytes())

	var person2 Person
	de := serdew.NewDeserializer(ser.Bytes())
	if err := SerdewPerson(de, &person2); err != nil {
		log.Fatalln(de.Error())
	}
	fmt.Println("Deserialized person:", person2)
}
```

The output of the example looks like this:
```
Pre-serialized person: {James 25 1.7 [123 Fake Rd] [{Mary 26 1.5 [Mary's House] [] map[] 2024-02-20 17:13:08.419104736 -0500 EST m=-86399.999978385} {Bob 45 1.6 [628 Apt. Ave Apartment 23] [{Nested Friend 84829340 +Inf [] [] map[] 0001-01-01 00:00:00 +0000 UTC}] map[] 2024-02-21 16:13:08.419105005 -0500 EST m=-3599.999978128}] map[test1:1 test2:29847329] 2024-02-21 17:13:08.419105106 -0500 EST m=+0.000021972}
Serialized bytes: [5 0 0 0 0 0 0 0 74 97 109 101 115 25 0 0 0 0 0 0 0 51 51 51 51 51 51 251 63 1 0 0 0 0 0 0 0 11 0 0 0 0 0 0 0 49 50 51 32 70 97 107 101 32 82 100 2 0 0 0 0 0 0 0 4 0 0 0 0 0 0 0 77 97 114 121 26 0 0 0 0 0 0 0 0 0 0 0 0 0 248 63 1 0 0 0 0 0 0 0 12 0 0 0 0 0 0 0 77 97 114 121 39 115 32 72 111 117 115 101 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 15 0 0 0 0 0 0 0 1 0 0 0 14 221 103 26 244 24 251 7 224 254 212 3 0 0 0 0 0 0 0 66 111 98 45 0 0 0 0 0 0 0 154 153 153 153 153 153 249 63 2 0 0 0 0 0 0 0 12 0 0 0 0 0 0 0 54 50 56 32 65 112 116 46 32 65 118 101 12 0 0 0 0 0 0 0 65 112 97 114 116 109 101 110 116 32 50 51 1 0 0 0 0 0 0 0 13 0 0 0 0 0 0 0 78 101 115 116 101 100 32 70 114 105 101 110 100 156 100 14 5 0 0 0 0 0 0 0 0 0 0 240 127 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 15 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0 0 0 0 255 255 0 0 0 0 0 0 0 0 15 0 0 0 0 0 0 0 1 0 0 0 14 221 104 94 100 24 251 8 237 254 212 2 0 0 0 0 0 0 0 5 0 0 0 0 0 0 0 116 101 115 116 49 1 0 0 0 0 0 0 0 5 0 0 0 0 0 0 0 116 101 115 116 50 33 111 199 1 0 0 0 0 15 0 0 0 0 0 0 0 1 0 0 0 14 221 104 108 116 24 251 9 82 254 212]
Deserialized person: {James 25 1.7 [123 Fake Rd] [{Mary 26 1.5 [Mary's House] [] map[] 2024-02-20 17:13:08.419104736 -0500 EST} {Bob 45 1.6 [628 Apt. Ave Apartment 23] [{Nested Friend 84829340 +Inf [] [] map[] 0001-01-01 00:00:00 +0000 UTC}] map[] 2024-02-21 16:13:08.419105005 -0500 EST}] map[test1:1 test2:29847329] 2024-02-21 17:13:08.419105106 -0500 EST}
```

## Error handling
If you read the above example closely, you might be curious about error handling. In Serdew, de/serializers are required to call `ctx.Abort(err)` whenever they encounter an error. After this function is called, the de/serializer is poisoned, and all future de/serialization operations will return an error. Because of this, user de/serializers can avoid the verbose `if err != nil { return err }` pattern except for if they wish to *slightly* improve performance in error conditions by exiting early.
