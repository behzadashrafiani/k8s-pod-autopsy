package render

import "github.com/charmbracelet/lipgloss"

type Styler interface {
	Heading(string) string
	Danger(string) string
	Dim(string) string
	Plain() bool
}

type plain struct{}

func PlainStyler() Styler              { return plain{} }
func (plain) Heading(s string) string { return s }
func (plain) Danger(s string) string  { return s }
func (plain) Dim(s string) string     { return s }
func (plain) Plain() bool             { return true }

type colored struct {
	heading, danger, dim lipgloss.Style
}

func ColorStyler() Styler {
	return colored{
		heading: lipgloss.NewStyle().Bold(true),
		danger:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),
		dim:     lipgloss.NewStyle().Faint(true),
	}
}
func (c colored) Heading(s string) string { return c.heading.Render(s) }
func (c colored) Danger(s string) string  { return c.danger.Render(s) }
func (c colored) Dim(s string) string     { return c.dim.Render(s) }
func (c colored) Plain() bool             { return false }
