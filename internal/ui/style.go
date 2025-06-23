package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var boxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(1).
	Width(30).
	Height(10).
	BorderForeground(lipgloss.Color("62"))

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Width(30).
	Align(lipgloss.Center).
	Foreground(lipgloss.Color("205"))

var PcTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("226")).
	Bold(true)
