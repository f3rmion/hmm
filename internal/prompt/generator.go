// Package prompt generates image prompts for HMM mnemonic scenes.
package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/f3rmion/hmm/internal/hmm"
)

// Generator creates image prompts from HMM scene data.
type Generator struct {
	actors   map[string]*hmm.Actor
	sets     map[string]*hmm.Set
	props    map[string]*hmm.Prop
	template *template.Template
	style    Style
}

// Style configures the image generation output.
type Style struct {
	Name        string // e.g., "photorealistic", "anime", "watercolor"
	AspectRatio string // e.g., "16:9", "1:1"
	Quality     string // e.g., "hd", "standard"
	Suffix      string // Added to end of prompt
	Negative    string // Negative prompt (for SD)
}

// DefaultStyle returns sensible defaults for image generation.
func DefaultStyle() Style {
	return Style{
		Name:        "cinematic digital art",
		AspectRatio: "16:9",
		Quality:     "hd",
		Suffix:      "dramatic lighting, detailed, memorable scene, mnemonic visualization",
		Negative:    "",
	}
}

// SceneData holds all the resolved data for generating a prompt.
type SceneData struct {
	Character   string
	Pinyin      string
	Meaning     string
	Tone        int
	ToneRoom    string
	Actor       *hmm.Actor
	Set         *hmm.Set
	Props       []*hmm.Prop
	Components  []string
	Style       Style
	Etymology   string
	Decomp      string
}

// NewGenerator creates a new prompt generator.
func NewGenerator(actors []hmm.Actor, sets []hmm.Set, props []hmm.Prop) *Generator {
	g := &Generator{
		actors: make(map[string]*hmm.Actor),
		sets:   make(map[string]*hmm.Set),
		props:  make(map[string]*hmm.Prop),
		style:  DefaultStyle(),
	}

	for i := range actors {
		g.actors[actors[i].ID] = &actors[i]
	}
	for i := range sets {
		g.sets[sets[i].ID] = &sets[i]
	}
	for i := range props {
		g.props[props[i].ID] = &props[i]
	}

	// Default template
	g.template = template.Must(template.New("prompt").Parse(defaultTemplate))

	return g
}

// SetStyle updates the image generation style.
func (g *Generator) SetStyle(style Style) {
	g.style = style
}

// SetTemplate sets a custom prompt template.
func (g *Generator) SetTemplate(tmpl string) error {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}
	g.template = t
	return nil
}

// GetActor returns the actor for a given initial.
func (g *Generator) GetActor(actorID string) *hmm.Actor {
	return g.actors[actorID]
}

// GetSet returns the set for a given final.
func (g *Generator) GetSet(setID string) *hmm.Set {
	return g.sets[setID]
}

// GetProp returns the prop for a given component.
func (g *Generator) GetProp(component string) *hmm.Prop {
	return g.props[component]
}

// GetToneRoom returns the room description for a tone within a set.
func (g *Generator) GetToneRoom(set *hmm.Set, tone hmm.Tone) string {
	if set == nil {
		return getToneRoomDefault(tone)
	}
	for _, room := range set.Rooms {
		if room.Tone == tone {
			if room.Description != "" {
				return room.Description
			}
			return room.Name
		}
	}
	return getToneRoomDefault(tone)
}

func getToneRoomDefault(tone hmm.Tone) string {
	switch tone {
	case hmm.Tone1:
		return "outside the entrance"
	case hmm.Tone2:
		return "in the kitchen"
	case hmm.Tone3:
		return "in the bedroom"
	case hmm.Tone4:
		return "in the bathroom"
	case hmm.Tone5:
		return "on the roof"
	default:
		return "inside"
	}
}

// Generate creates an image prompt for a character scene.
func (g *Generator) Generate(data SceneData) (string, error) {
	data.Style = g.style

	var buf bytes.Buffer
	if err := g.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// GenerateSimple creates a simple descriptive prompt without templates.
func (g *Generator) GenerateSimple(data SceneData) string {
	var parts []string

	// Actor description
	if data.Actor != nil && data.Actor.Name != "" {
		parts = append(parts, data.Actor.Name)
	} else {
		parts = append(parts, "A person")
	}

	// Location
	if data.Set != nil && data.Set.Name != "" {
		parts = append(parts, fmt.Sprintf("at %s", data.Set.Name))
	}

	// Tone room
	if data.ToneRoom != "" {
		parts = append(parts, fmt.Sprintf("(%s)", data.ToneRoom))
	}

	// Props
	if len(data.Props) > 0 {
		propNames := make([]string, 0, len(data.Props))
		for _, p := range data.Props {
			if p != nil && p.Name != "" {
				propNames = append(propNames, p.Name)
			}
		}
		if len(propNames) > 0 {
			parts = append(parts, fmt.Sprintf("with %s", strings.Join(propNames, " and ")))
		}
	}

	// Meaning context
	if data.Meaning != "" {
		parts = append(parts, fmt.Sprintf("representing '%s'", data.Meaning))
	}

	prompt := strings.Join(parts, " ")

	// Add style suffix
	if g.style.Suffix != "" {
		prompt += ", " + g.style.Suffix
	}

	return prompt
}

// BuildSceneData constructs SceneData from HMM components.
func (g *Generator) BuildSceneData(
	character string,
	pinyin string,
	actorID string,
	setID string,
	tone hmm.Tone,
	components []string,
	meaning string,
	etymology string,
	decomp string,
) SceneData {
	actor := g.GetActor(actorID)
	set := g.GetSet(setID)

	var props []*hmm.Prop
	for _, comp := range components {
		if p := g.GetProp(comp); p != nil {
			props = append(props, p)
		}
	}

	return SceneData{
		Character:  character,
		Pinyin:     pinyin,
		Meaning:    meaning,
		Tone:       int(tone),
		ToneRoom:   g.GetToneRoom(set, tone),
		Actor:      actor,
		Set:        set,
		Props:      props,
		Components: components,
		Etymology:  etymology,
		Decomp:     decomp,
	}
}

// Default prompt template
const defaultTemplate = `{{- /* HMM Image Prompt Template */ -}}
{{- if .Actor }}{{if .Actor.Name}}{{ .Actor.Name }}{{else}}A person{{end}}{{else}}A person{{end}}
{{- if .Set }}{{if .Set.Name}} at {{ .Set.Name }}{{end}}{{end}}
{{- if .ToneRoom }} ({{ .ToneRoom }}){{end}}
{{- if .Props }}, interacting with {{ range $i, $p := .Props }}{{if $i}} and {{end}}{{if $p.Name}}{{ $p.Name }}{{else}}{{ $p.Component }}{{end}}{{ end }}{{end}}
{{- if .Meaning }}, scene represents "{{ .Meaning }}"{{end}}
{{- if .Etymology }}, etymology: {{ .Etymology }}{{end}}.
{{ .Style.Name }}, {{ .Style.Suffix }}`

// MidjourneyTemplate is optimized for Midjourney.
const MidjourneyTemplate = `{{- if .Actor }}{{if .Actor.Name}}{{ .Actor.Name }}{{else}}person{{end}}{{else}}person{{end}}
{{- if .Set }}{{if .Set.Name}} in {{ .Set.Name }}{{end}}{{end}}
{{- if .ToneRoom }}, {{ .ToneRoom }} area{{end}}
{{- if .Props }}, holding {{ range $i, $p := .Props }}{{if $i}}, {{end}}{{if $p.Name}}{{ $p.Name }}{{else}}{{ $p.Component }}{{end}}{{ end }}{{end}}
{{- if .Meaning }}, representing {{ .Meaning }}{{end}}
--ar {{ .Style.AspectRatio }} --v 6 --style raw`

// DALLETemplate is optimized for DALL-E.
const DALLETemplate = `A {{ .Style.Name }} scene: {{- if .Actor }}{{if .Actor.Name}} {{ .Actor.Name }}{{else}} a person{{end}}{{else}} a person{{end}}
{{- if .Set }}{{if .Set.Name}} inside {{ .Set.Name }}{{end}}{{end}}
{{- if .ToneRoom }}, specifically {{ .ToneRoom }}{{end}}
{{- if .Props }}. They are interacting with {{ range $i, $p := .Props }}{{if $i}} and {{end}}{{if $p.Name}}a {{ $p.Name }}{{else}}{{ $p.Component }}{{end}}{{ end }}{{end}}
{{- if .Meaning }}. The scene symbolizes "{{ .Meaning }}"{{end}}.
{{ .Style.Suffix }}`

// StableDiffusionTemplate is optimized for Stable Diffusion.
const StableDiffusionTemplate = `{{- if .Actor }}{{if .Actor.Name}}({{ .Actor.Name }}:1.2){{else}}(person:1.1){{end}}{{else}}(person:1.1){{end}},
{{- if .Set }}{{if .Set.Name}}({{ .Set.Name }} interior:1.1){{end}}{{end}},
{{- if .ToneRoom }}{{ .ToneRoom }}, {{end}}
{{- if .Props }}{{ range $i, $p := .Props }}{{if $i}}, {{end}}({{if $p.Name}}{{ $p.Name }}{{else}}{{ $p.Component }}{{end}}:1.1){{ end }}, {{end}}
{{ .Style.Name }}, {{ .Style.Suffix }}, masterpiece, best quality`
