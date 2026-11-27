package supabank

import "testing"

func TestAssemble(t *testing.T) {
	pack, err := Assemble([]Row{
		{PromptID: "dog-breed", PromptText: "A dog breed", CategoryName: "Dog breeds", EntityID: "Q2", Label: "Poodle", Rarity: 40, Alias: "Poodle dog"},
		{PromptID: "dog-breed", PromptText: "A dog breed", CategoryName: "Dog breeds", EntityID: "Q1", Label: "Labrador Retriever", Rarity: 5, Alias: "Lab"},
		{PromptID: "dog-breed", PromptText: "A dog breed", CategoryName: "Dog breeds", EntityID: "Q1", Label: "Labrador Retriever", Rarity: 5, Alias: "Labrador"},
		{PromptID: "dog-breed", PromptText: "A dog breed", CategoryName: "Dog breeds", EntityID: "Q1", Label: "Labrador Retriever", Rarity: 5, Alias: "Lab"},
		{PromptID: "element", PromptText: "A chemical element", CategoryName: "Chemical elements", EntityID: "Q556", Label: "Hydrogen", Rarity: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.Prompts) != 2 {
		t.Fatalf("prompts = %d, want 2", len(pack.Prompts))
	}
	// Sorted by prompt id: dog-breed, then element.
	dog := pack.Prompts[0]
	if dog.ID != "dog-breed" || dog.Category != "Dog breeds" || dog.Text != "A dog breed" {
		t.Fatalf("dog prompt = %+v", dog)
	}
	if len(dog.Entries) != 2 || dog.Entries[0].ID != "Q1" || dog.Entries[1].ID != "Q2" {
		t.Fatalf("entries = %+v", dog.Entries)
	}
	lab := dog.Entries[0]
	if lab.Label != "Labrador Retriever" || lab.Rarity != 5 {
		t.Fatalf("lab = %+v", lab)
	}
	if len(lab.Aliases) != 2 || lab.Aliases[0] != "Lab" || lab.Aliases[1] != "Labrador" {
		t.Fatalf("aliases = %v", lab.Aliases)
	}
	el := pack.Prompts[1]
	if el.ID != "element" || len(el.Entries) != 1 || el.Entries[0].Label != "Hydrogen" || el.Entries[0].Rarity != 1 {
		t.Fatalf("element = %+v", el)
	}
	if len(el.Entries[0].Aliases) != 0 {
		t.Fatalf("hydrogen aliases = %v", el.Entries[0].Aliases)
	}
}
