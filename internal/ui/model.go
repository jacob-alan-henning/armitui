package ui

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jacob-alan-henning/armitui/internal/arm"
	"github.com/jacob-alan-henning/armitui/internal/macho"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	cpu                arm.ARM64CPU
	instructionHistory []string
	progEnded          bool
}

func InitialModel() model {
	return model{
		cpu:                arm.NewArm64CPU(),
		instructionHistory: make([]string, 0),
		progEnded:          false,
	}
}

type loadProgramMsg struct{ asmFile string }

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		return loadProgramMsg{"asm/demo"}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadProgramMsg:
		rawProgBytes, err := macho.GetTextSectionBytes("asm/demo")
		if err != nil {
			log.Fatalf("could not load binary into memory %v", err)
		}
		m.cpu.LoadProgram(rawProgBytes)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			if !m.progEnded {
				m.instructionHistory = append(m.instructionHistory, m.cpu.GetCurrentInst())

				err := m.cpu.Step()
				if err != nil {
					m.progEnded = true
					return m, nil
				}
			}
		}
	}
	return m, nil
}

func (m model) buildInstructionHistory() string {
	var stateBuilder strings.Builder
	curr := len(m.instructionHistory) - 1
	for i := range m.instructionHistory {
		if curr != i {
			fmt.Fprintf(&stateBuilder, "   %s\n", m.instructionHistory[i])
		} else {
			fmt.Fprintf(&stateBuilder, "*  %s\n", m.instructionHistory[i])
		}
	}
	return stateBuilder.String()
}

func (m model) View() string {
	titleContent := titleStyle.
		Foreground(lipgloss.Color("20")).
		Align(lipgloss.Left).
		Height(1).
		Render("ARMITUI - intuitive ARM stepper")

	regContent := titleStyle.Render("Registers") + "\n" + boxStyle.Render(m.cpu.BuildGPRegisterstate())
	instContent := titleStyle.Render("Instruction History") + "\n" + boxStyle.Render(m.buildInstructionHistory())
	memContent := titleStyle.Render("Memory") + "\n" + boxStyle.Width(70).Render(m.cpu.BuildMemoryState())

	cmd := titleStyle.Render("Commands") + "\n" + boxStyle.Render("(q)uit \n(s)tep")

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, regContent, instContent, memContent)

	fullScreen := lipgloss.JoinVertical(lipgloss.Left, titleContent, mainArea, cmd)

	return lipgloss.Place(70, 15, lipgloss.Center, lipgloss.Center, fullScreen)
}

func StartUI() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
