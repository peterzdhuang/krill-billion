package data

// Pack is a versioned question pack loaded from JSON.
type Pack struct {
	PackID      string   `json:"pack_id"`
	Version     int      `json:"version"`
	BuiltAt     string   `json:"built_at"`
	Attribution []string `json:"attribution"`
	Prompts     []Prompt `json:"prompts"`
}

// Prompt is a closed-category rarity-guessing question.
type Prompt struct {
	ID       string  `json:"id"`
	Category string  `json:"category"`
	Text     string  `json:"text"`
	Entries  []Entry `json:"entries"`
}

// Entry is a valid answer with rarity 0–100 (higher = more niche).
type Entry struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases"`
	Rarity  int      `json:"rarity"`
}
