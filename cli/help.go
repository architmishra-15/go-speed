package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	version     = "v1.0.0"
	projectName = "SpeedTest CLI"
	description = "A fast and beautiful network speed testing tool for the terminal"
	license     = "Apache 2.0"
	author      = "Archit Mishra"
	githubRepo  = "https://github.com/architmishra-15/speedtest-cli"
)

// Color palette
var (
	primaryColor   = lipgloss.Color("39")  // Blue
	secondaryColor = lipgloss.Color("46")  // Green
	accentColor    = lipgloss.Color("214") // Orange
	textColor      = lipgloss.Color("255") // White
	mutedColor     = lipgloss.Color("243") // Gray
	errorColor     = lipgloss.Color("196") // Red
)

// Styles
var (
	// Main container
	containerStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Margin(1, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(lipgloss.Color("236")).
			Padding(1, 2).
			MarginBottom(1).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	// Description style
	descriptionStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				MarginBottom(2).
				Align(lipgloss.Center)

	// Section header style
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor).
				MarginTop(1).
				MarginBottom(1)

	// Info item style
	infoLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Width(12)

	infoValueStyle = lipgloss.NewStyle().
			Foreground(textColor).
			MarginLeft(1)

	// Link style
	linkStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Underline(true)

	// Usage style
	usageStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Background(lipgloss.Color("237")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	// Command style
	commandStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Width(12)

	commandDescStyle = lipgloss.NewStyle().
				Foreground(textColor).
				MarginLeft(1)

	// Version style
	versionStyle = lipgloss.NewStyle().
			Bold(true)
	// Foreground(secondaryColor).
	// Background(lipgloss.Color("238")).
	// Padding(0, 1)
	// Border(lipgloss.RoundedBorder()).
	// BorderForeground(secondaryColor)
)

type modelmsg struct {
	mode string // "help" or "version"
}

func initialModelMsg(mode string) modelmsg {
	return modelmsg{mode: mode}
}

func (m modelmsg) Init() tea.Cmd {
	return tea.Quit
}

func (m modelmsg) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m modelmsg) View() string {
	if m.mode == "help" {
		return m.renderHelp()
	} else if m.mode == "version" {
		return m.renderVersion()
	}
	return ""
}

func (m modelmsg) renderHelp() string {
	var sections []string

	// Title
	sections = append(sections, titleStyle.Render(projectName))

	// Description
	sections = append(sections, descriptionStyle.Render(description))

	// Project info
	sections = append(sections, sectionHeaderStyle.Render("Project Information:"))
	infoContent := strings.Join([]string{
		lipgloss.JoinHorizontal(lipgloss.Top, infoLabelStyle.Render("Author:"), infoValueStyle.Render(author)),
		lipgloss.JoinHorizontal(lipgloss.Top, infoLabelStyle.Render("License:"), infoValueStyle.Render(license)),
		lipgloss.JoinHorizontal(lipgloss.Top, infoLabelStyle.Render("Version:"), infoValueStyle.Render(versionStyle.Render(version))),
		lipgloss.JoinHorizontal(lipgloss.Top, infoLabelStyle.Render("Repository:"), infoValueStyle.Render(linkStyle.Render(githubRepo))),
	}, "\n")
	sections = append(sections, infoContent)

	// Usage
	sections = append(sections, sectionHeaderStyle.Render("Usage:"))
	sections = append(sections, usageStyle.Render("speedtest [command]"))

	// Available commands
	sections = append(sections, sectionHeaderStyle.Render("Available Commands:"))
	commandsContent := strings.Join([]string{
		lipgloss.JoinHorizontal(lipgloss.Top, commandStyle.Render("help"), commandDescStyle.Render("Show this help message")),
		lipgloss.JoinHorizontal(lipgloss.Top, commandStyle.Render("version"), commandDescStyle.Render("Show version information")),
		lipgloss.JoinHorizontal(lipgloss.Top, commandStyle.Render("test"), commandDescStyle.Render("Run network speed test")),
		lipgloss.JoinHorizontal(lipgloss.Top, commandStyle.Render("servers"), commandDescStyle.Render("List available test servers")),
	}, "\n")
	sections = append(sections, commandsContent)

	return containerStyle.Render(strings.Join(sections, "\n"))
}

func (m modelmsg) renderVersion() string {
	// Simple version display
	versionContent := lipgloss.NewStyle().Render(versionStyle.Render(version))
	// titleStyle.Width(0).Render(projectName),
	// lipgloss.NewStyle().MarginLeft(1).Render("•"),

	return containerStyle.Width(50).Render(versionContent)
}
