# HMM - Hanzi Movie Method

A CLI tool for learning Chinese characters using the Hanzi Movie Method mnemonic system.

## What is the Hanzi Movie Method?

The Hanzi Movie Method is a powerful memorization technique that transforms each Chinese character into a vivid, memorable movie scene. It works by mapping the components of a character's pronunciation and structure to visual elements:

| Element | Maps To | Purpose |
|---------|---------|---------|
| Pinyin Initial | Actor (person) | 55 unique actors for each initial sound |
| Pinyin Final | Set (location) | 38 unique locations for each final sound |
| Tone | Room (area within set) | 5 rooms per location for tones 1-5 |
| Components | Props (objects) | 214 radicals as memorable objects |

For example, the character 好 (hǎo, "good"):
- Initial `h` → Your chosen actor (e.g., Harrison Ford)
- Final `ao` → Your chosen location (e.g., an Office)
- Tone `3` → A specific room in that location
- Components `女 + 子` → Props: a red dress + a baby rattle

You imagine Harrison Ford in an office, interacting with a red dress and baby rattle - creating a unique, memorable scene for this character.

## Installation

```bash
go install github.com/f3rmion/hmm/cmd/hmm@latest
```

Or build from source:

```bash
git clone https://github.com/f3rmion/hmm
cd hmm
go build -o hmm ./cmd/hmm
```

## Usage

### Interactive TUI

Simply run `hmm` to launch the interactive terminal UI:

```bash
hmm
```

The TUI provides:
- Lookup View (1) - Type characters to see their HMM breakdown
- Browse View (2) - Browse Anki deck cards with HMM data
- Open Deck (3) - Load an Anki .apkg file
- Settings (4) - View your configuration

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `1-4` | Switch views |
| `Tab` | Toggle sidebar focus |
| `?` | Show help (when sidebar focused) |
| `q` | Quit |

Lookup View:

| Key | Action |
|-----|--------|
| `Enter` | Analyze character(s) |
| `g` | Generate LLM prompt |
| `y` | Copy prompt to clipboard |
| `←/→` | Navigate between characters |

Browse View:

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate cards |
| `←/→` | Navigate characters in card |
| `/` | Search |
| `g` | Generate prompt for current |
| `B` | Batch generate all prompts |

### CLI Commands

```bash
# Look up a character
hmm lookup 好

# Generate an image prompt
hmm generate 好 --verbose

# Generate with different AI art styles
hmm generate 林 --style midjourney
hmm generate 中 --style dalle
hmm generate 水 --style sd

# Inspect an Anki deck
hmm anki inspect deck.apkg

# Augment Anki deck with HMM data
hmm anki augment deck.apkg --output augmented.json
```

## Configuration

On first run, HMM creates configuration files in `~/.config/hmm/`:

```
~/.config/hmm/
├── actors.yaml    # Your 55 actors (pinyin initials)
├── sets.yaml      # Your 38 locations (pinyin finals)
├── props.yaml     # Your 214+ props (radicals/components)
└── anki/          # Anki decks
```

### Personalizing Your System

The key to the Hanzi Movie Method is personal connections. Edit the config files to use:

- Actors: People you know well (friends, family, celebrities)
- Sets: Places you can vividly imagine (your home, workplace, favorite spots)
- Props: Objects that resonate with you

Example `actors.yaml`:
```yaml
actors:
  - id: "h"
    initial: "h"
    name: "Harrison Ford"
    description: "Indiana Jones himself"
```

Example `sets.yaml`:
```yaml
sets:
  - id: "ao"
    final: "ao"
    name: "Office, Berlin"
    rooms:
      tone1: "Reception desk"
      tone2: "Conference room"
      tone3: "Your workspace"
      tone4: "Break room"
      tone5: "Storage closet"
```

Example `props.yaml`:
```yaml
props:
  - id: "女"
    component: "女"
    name: "Red dress"
    meaning: "woman, female"
    description: "A beautiful flowing red dress"
```

## Props: The 214 Kangxi Radicals

HMM includes all 214 Kangxi radicals as props, organized into categories:

- Basic Strokes - 一 (chopstick), 丨 (bamboo pole), 丶 (water droplet)
- People & Body - 人 (stick figure), 口 (megaphone), 心 (valentine heart)
- Nature - 日 (flashlight), 水 (water bottle), 火 (torch), 山 (mountain peak)
- Animals - 犬 (dog), 馬 (saddle), 鳥 (bird cage), 魚 (fishing rod)
- Objects - 刀 (chef's knife), 車 (wheel), 門 (gate)
- Actions & Concepts - 走 (running track), 言 (microphone), 力 (dumbbell)

Each radical has a memorable prop name that can be visualized in your movie scenes.

## LLM Integration

HMM can generate detailed scene descriptions using Claude AI. Set your API key:

```bash
export ANTHROPIC_API_KEY=your-key-here
```

Then press `g` in the TUI to generate a vivid scene description, or `y` to copy it to clipboard.

## Data Sources

- Character decomposition data from [Make Me a Hanzi](https://github.com/skishore/makemeahanzi)
- Pinyin data from standard Chinese dictionaries
- 214 Kangxi radicals with traditional meanings

## License

MIT
