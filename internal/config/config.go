// Package config handles loading and saving user configuration for HMM.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/f3rmion/hmm/internal/hmm"
	"gopkg.in/yaml.v3"
)

// Config holds all user configuration for the HMM system.
type Config struct {
	Actors []hmm.Actor `yaml:"actors"`
	Sets   []hmm.Set   `yaml:"sets"`
	Props  []hmm.Prop  `yaml:"props"`
}

// PromptConfig holds settings for image prompt generation.
type PromptConfig struct {
	Style       string `yaml:"style"`        // e.g., "photorealistic", "anime", "watercolor"
	AspectRatio string `yaml:"aspect_ratio"` // e.g., "16:9", "1:1", "4:3"
	Quality     string `yaml:"quality"`      // e.g., "standard", "hd"
	Negative    string `yaml:"negative"`     // Negative prompt (things to avoid)
	Suffix      string `yaml:"suffix"`       // Added to end of every prompt
}

// LoadActors loads actors configuration from a YAML file.
func LoadActors(path string) ([]hmm.Actor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading actors file: %w", err)
	}

	var actors struct {
		Actors []hmm.Actor `yaml:"actors"`
	}
	if err := yaml.Unmarshal(data, &actors); err != nil {
		return nil, fmt.Errorf("parsing actors file: %w", err)
	}

	return actors.Actors, nil
}

// LoadSets loads sets configuration from a YAML file.
func LoadSets(path string) ([]hmm.Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading sets file: %w", err)
	}

	var sets struct {
		Sets []hmm.Set `yaml:"sets"`
	}
	if err := yaml.Unmarshal(data, &sets); err != nil {
		return nil, fmt.Errorf("parsing sets file: %w", err)
	}

	return sets.Sets, nil
}

// LoadProps loads props configuration from a YAML file.
func LoadProps(path string) ([]hmm.Prop, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading props file: %w", err)
	}

	var props struct {
		Props []hmm.Prop `yaml:"props"`
	}
	if err := yaml.Unmarshal(data, &props); err != nil {
		return nil, fmt.Errorf("parsing props file: %w", err)
	}

	return props.Props, nil
}

// LoadConfig loads all configuration from a directory.
func LoadConfig(dir string) (*Config, error) {
	actors, err := LoadActors(filepath.Join(dir, "actors.yaml"))
	if err != nil {
		return nil, err
	}

	sets, err := LoadSets(filepath.Join(dir, "sets.yaml"))
	if err != nil {
		return nil, err
	}

	props, err := LoadProps(filepath.Join(dir, "props.yaml"))
	if err != nil {
		return nil, err
	}

	return &Config{
		Actors: actors,
		Sets:   sets,
		Props:  props,
	}, nil
}

// SaveActors saves actors configuration to a YAML file.
func SaveActors(path string, actors []hmm.Actor) error {
	data := struct {
		Actors []hmm.Actor `yaml:"actors"`
	}{Actors: actors}

	out, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("marshaling actors: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing actors file: %w", err)
	}

	return nil
}

// SaveSets saves sets configuration to a YAML file.
func SaveSets(path string, sets []hmm.Set) error {
	data := struct {
		Sets []hmm.Set `yaml:"sets"`
	}{Sets: sets}

	out, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("marshaling sets: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing sets file: %w", err)
	}

	return nil
}

// SaveProps saves props configuration to a YAML file.
func SaveProps(path string, props []hmm.Prop) error {
	data := struct {
		Props []hmm.Prop `yaml:"props"`
	}{Props: props}

	out, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("marshaling props: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing props file: %w", err)
	}

	return nil
}

// GetConfigDir returns the default configuration directory.
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hmm"), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
