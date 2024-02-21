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
