// Package hmm provides core types and logic for the Hanzi Movie Method.
package hmm

// ActorCategory represents the four categories of actors in HMM.
type ActorCategory string

const (
	ActorMale       ActorCategory = "male"       // Real men - basic consonant initials
	ActorFemale     ActorCategory = "female"     // Real women - consonant+i initials
	ActorFictional  ActorCategory = "fictional"  // Fictional characters - consonant+u initials
	ActorGodLeader  ActorCategory = "god_leader" // Gods or world leaders - consonant+ü initials
	ActorNull       ActorCategory = "null"       // Null initial (no consonant)
)

// Actor represents a person (real or fictional) mapped to a pinyin initial.
// The actor's name should give a clue to the pronunciation.
type Actor struct {
	ID          string        `yaml:"id" json:"id"`                     // Unique identifier (e.g., "b", "bi", "bu")
	Initial     string        `yaml:"initial" json:"initial"`           // The pinyin initial this actor represents
	Category    ActorCategory `yaml:"category" json:"category"`         // male, female, fictional, god_leader, null
	Name        string        `yaml:"name" json:"name"`                 // The actor's name (e.g., "Brad Pitt")
	Description string        `yaml:"description,omitempty" json:"description,omitempty"` // Optional description or notes
	ImagePrompt string        `yaml:"image_prompt,omitempty" json:"image_prompt,omitempty"` // Description for image generation
}

// Tone represents the four tones of Mandarin plus neutral tone.
type Tone int

const (
	Tone1      Tone = 1 // First tone (high level) - ˉ
	Tone2      Tone = 2 // Second tone (rising) - ˊ
	Tone3      Tone = 3 // Third tone (dipping) - ˇ
	Tone4      Tone = 4 // Fourth tone (falling) - ˋ
	Tone5      Tone = 5 // Fifth tone (neutral)
	ToneUnknown Tone = 0
)

// ToneRoom represents a specific area within a Set that corresponds to a tone.
type ToneRoom struct {
	Tone        Tone   `yaml:"tone" json:"tone"`
	Name        string `yaml:"name" json:"name"`               // e.g., "entrance", "kitchen", "bedroom"
	Description string `yaml:"description,omitempty" json:"description,omitempty"` // Personal description of this area
	ImagePrompt string `yaml:"image_prompt,omitempty" json:"image_prompt,omitempty"` // Description for image generation
}

// Set represents a location (memory palace) mapped to a pinyin final.
type Set struct {
	ID          string     `yaml:"id" json:"id"`                     // Unique identifier (e.g., "a", "ao", "ang")
	Final       string     `yaml:"final" json:"final"`               // The pinyin final this set represents
	Name        string     `yaml:"name" json:"name"`                 // The location name (e.g., "Childhood Home")
	Link        string     `yaml:"link,omitempty" json:"link,omitempty"` // How this location links to the final sound
	Description string     `yaml:"description,omitempty" json:"description,omitempty"` // Personal description/memories
	Epoch       string     `yaml:"epoch,omitempty" json:"epoch,omitempty"` // Life chapter this location represents
	Rooms       []ToneRoom `yaml:"rooms" json:"rooms"`               // The 5 tone rooms within this set
	ImagePrompt string     `yaml:"image_prompt,omitempty" json:"image_prompt,omitempty"` // Description for image generation
}

// PropType indicates how the prop relates to the component.
type PropType string

const (
	PropAppearance  PropType = "appearance"  // Based on what the component looks like
	PropMeaning     PropType = "meaning"     // Based on the component's meaning
	PropCombination PropType = "combination" // Based on both appearance and meaning
)

// Prop represents an object mapped to a character component/radical.
type Prop struct {
	ID          string   `yaml:"id" json:"id"`                     // Unique identifier (the component itself, e.g., "木", "口")
	Component   string   `yaml:"component" json:"component"`       // The character component
	Name        string   `yaml:"name" json:"name"`                 // The prop object (e.g., "tree", "mouth/opening")
	Type        PropType `yaml:"type,omitempty" json:"type,omitempty"` // How the prop relates to component
	Meaning     string   `yaml:"meaning,omitempty" json:"meaning,omitempty"` // Original meaning of the component
	Description string   `yaml:"description,omitempty" json:"description,omitempty"` // Why this prop was chosen
	ImagePrompt string   `yaml:"image_prompt,omitempty" json:"image_prompt,omitempty"` // Description for image generation
}

// Etymology holds information about a character's origin.
type Etymology struct {
	Type     string `json:"type"`               // pictophonetic, pictographic, ideographic
	Semantic string `json:"semantic,omitempty"` // Meaning component (for pictophonetic)
	Phonetic string `json:"phonetic,omitempty"` // Sound component (for pictophonetic)
	Hint     string `json:"hint,omitempty"`     // Additional etymology hint
}

// HanziEntry represents a Chinese character with all its data.
type HanziEntry struct {
	Character     string     `json:"character"`
	Pinyin        []string   `json:"pinyin"`        // All possible readings
	Definition    string     `json:"definition"`    // English meaning(s)
	Decomposition string     `json:"decomposition"` // IDS decomposition string
	Components    []string   `json:"components"`    // Individual components
	Radical       string     `json:"radical"`       // Kangxi radical
	Etymology     *Etymology `json:"etymology,omitempty"`
	StrokeCount   int        `json:"stroke_count,omitempty"`
}

// Scene represents a complete HMM mnemonic scene for a character.
type Scene struct {
	Character   string   `yaml:"character" json:"character"`
	Pinyin      string   `yaml:"pinyin" json:"pinyin"`           // Selected reading
	Initial     string   `yaml:"initial" json:"initial"`         // Extracted initial
	Final       string   `yaml:"final" json:"final"`             // Extracted final
	Tone        Tone     `yaml:"tone" json:"tone"`               // Extracted tone
	Keyword     string   `yaml:"keyword" json:"keyword"`         // The meaning/keyword to remember
	ActorID     string   `yaml:"actor_id" json:"actor_id"`       // Reference to actor
	SetID       string   `yaml:"set_id" json:"set_id"`           // Reference to set
	PropIDs     []string `yaml:"prop_ids" json:"prop_ids"`       // References to props
	Script      string   `yaml:"script" json:"script"`           // The mnemonic story
	ImagePrompt string   `yaml:"image_prompt,omitempty" json:"image_prompt,omitempty"` // Full prompt for image generation
}

// SpecialEffect represents a memory enhancement technique.
type SpecialEffect string

const (
	EffectSlowMotion    SpecialEffect = "slow_motion"    // Bullet time
	EffectDifferentAngle SpecialEffect = "different_angle" // Camera angle changes
	EffectBrightShining SpecialEffect = "bright_shining" // Make element prominent
	EffectSoundEffects  SpecialEffect = "sound_effects"  // Add sounds
	EffectThemeMusic    SpecialEffect = "theme_music"    // Add background music
	EffectExplosion     SpecialEffect = "explosion"      // Dramatic explosion
	EffectContrast      SpecialEffect = "contrast"       // Show before/after
	EffectReaction      SpecialEffect = "reaction"       // Emphasize reactions
	EffectHumor         SpecialEffect = "humor"          // Add comedic element
)
