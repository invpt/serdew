package main

import (
	"fmt"
	"math"

	"github.com/invpt/serdew"
)

type Person struct {
	Name       string
	Age        int
	Height     float64
	Address    []string
	Friends    []Person
	ExampleMap map[string]int
}

func SerdewPerson[S serdew.SerDe](ctx *serdew.Ctx[S], p *Person) {
	serdew.String(ctx, &p.Name)
	serdew.Number(ctx, &p.Age)
	serdew.Number(ctx, &p.Height)
	serdew.Slice(ctx, &p.Address, serdew.String)
	serdew.Slice(ctx, &p.Friends, SerdewPerson)
	serdew.Map(ctx, &p.ExampleMap, serdew.String, serdew.Number)
}

func main() {
	person1 := Person{
		Name:    "James",
		Age:     25,
		Height:  1.7,
		Address: []string{"123 Fake Rd"},
		Friends: []Person{
			{
				Name:    "Mary",
				Age:     26,
				Height:  1.5,
				Address: []string{"Mary's House"},
				Friends: []Person{},
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
			},
		},
		ExampleMap: map[string]int{
			"test1": 1,
			"test2": 29847329,
		},
	}

	fmt.Println("Pre-serialized person:", person1)

	ser := serdew.NewSerializer()
	SerdewPerson(ser, &person1)
	fmt.Println("Serialized bytes:", ser.Bytes())

	var person2 Person
	de := serdew.NewDeserializer(ser.Bytes())
	SerdewPerson(de, &person2)
	fmt.Println("Deserialized person:", person2)
}
