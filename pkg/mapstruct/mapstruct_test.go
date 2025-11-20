package mapstruct_test

import (
	"testing"

	"github.com/omalloc/tavern/pkg/mapstruct"
)

func TestDecode_SuccessAndNested(t *testing.T) {
	type Address struct {
		City string `json:"city"`
		Zip  int    `json:"zip"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Address Address `json:"address"`
	}

	input := map[string]interface{}{
		"name": "Alice",
		"age":  30,
		"address": map[string]interface{}{
			"city": "New York",
			"zip":  10001,
		},
	}

	var p Person
	if err := mapstruct.Decode(input, &p); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if p.Name != "Alice" {
		t.Fatalf("expected Name == %q, got %q", "Alice", p.Name)
	}
	if p.Age != 30 {
		t.Fatalf("expected Age == %d, got %d", 30, p.Age)
	}
	if p.Address.City != "New York" || p.Address.Zip != 10001 {
		t.Fatalf("unexpected Address: %+v", p.Address)
	}
}

func TestDecode_Slice(t *testing.T) {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	input := []map[string]interface{}{
		{"id": 1, "name": "one"},
		{"id": 2, "name": "two"},
	}

	var items []Item
	if err := mapstruct.Decode(input, &items); err != nil {
		t.Fatalf("Decode slice returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != 1 || items[0].Name != "one" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].ID != 2 || items[1].Name != "two" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}

func TestDecode_NonPointerOutputReturnsError(t *testing.T) {
	type Simple struct {
		Value string `json:"value"`
	}

	input := map[string]interface{}{"value": "x"}

	var s Simple
	// pass non-pointer output on purpose
	err := mapstruct.Decode(input, s)
	if err == nil {
		t.Fatalf("expected error when output is non-pointer, got nil")
	}
}
