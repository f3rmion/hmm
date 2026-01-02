// Package tui provides an interactive terminal UI for HMM.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#FF6B6B") // Red - titles, actors
	ColorSecondary = lipgloss.Color("#4ecdc4") // Teal - sets, subtitles
	ColorAccent    = lipgloss.Color("#ffe66d") // Yellow - characters, props
	ColorMuted     = lipgloss.Color("#666666") // Gray - help text
	ColorSuccess   = lipgloss.Color("#a8e6cf") // Green - success, tones
	ColorText      = lipgloss.Color("#f1faee") // Light text
	ColorLabel     = lipgloss.Color("#a8dadc") // Label color
	ColorBg        = lipgloss.Color("#1a1a2e") // Dark background
	ColorBgAlt     = lipgloss.Color("#2d3436") // Alt background
	ColorBorder    = lipgloss.Color("#3d5a80") // Border color
)

// Sidebar styles
var (
	SidebarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderRight(true).
			BorderForeground(ColorBorder).
			Padding(1, 1)

	SidebarTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Background(ColorBg).
				Padding(0, 1).
				MarginBottom(1)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)

	SidebarItemActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgAlt).
				Padding(0, 1)

	SidebarHelpStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				MarginTop(1).
				Padding(0, 1)
)

// Title styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBg).
			Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)
)

// Character display styles
var (
	CharacterLargeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgAlt).
				Padding(1, 4).
				Margin(1, 0).
				Align(lipgloss.Center)

	CharacterPinyinStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Italic(true).
				Align(lipgloss.Center)

	CharacterMeaningStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true).
				Align(lipgloss.Center)
)

// Character tab styles (for multi-character words)
var (
	CharTabStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 2).
			Margin(0, 1)

	CharTabActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgAlt).
				Padding(0, 2).
				Margin(0, 1)

	CharTabPinyinStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true)

	WordNavStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			Padding(0, 1)

	WordDisplayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Padding(0, 2).
				Margin(1, 0)
)

// HMM breakdown styles
var (
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorLabel).
			Bold(true).
			Width(12)

	ValueStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	ActorStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	SetStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	PropStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	ToneStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)
)

// Box styles
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	PromptBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(1, 2).
			Margin(1, 0)

	LLMPromptStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			Margin(1, 0)

	SearchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1)
)

// Status styles
var (
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	LoadingStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			Italic(true)

	CopiedStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)
)

// File picker styles
var (
	FilePickerDirStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	FilePickerFileStyle = lipgloss.NewStyle().
				Foreground(ColorText)

	FilePickerSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgAlt)

	FilePickerPathStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true)
)

// Settings view styles
var (
	SettingsTabStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 2)

	SettingsTabActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgAlt).
				Padding(0, 2)

	SettingsHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorLabel)

	SettingsRowStyle = lipgloss.NewStyle().
				Foreground(ColorText)
)

// Content area style
var ContentStyle = lipgloss.NewStyle().
	Padding(1, 2)

// Card count style (for browse view)
var CardCountStyle = lipgloss.NewStyle().
	Foreground(ColorMuted).
	Padding(0, 1)
