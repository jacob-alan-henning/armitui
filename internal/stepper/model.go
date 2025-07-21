package stepper

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/arch/arm64/arm64asm"
)

type model struct {
	// CPU execution state
	gpRegister   map[arm64asm.Reg]uint64
	mem          map[uint64]byte
	pc           int
	lastMemWrite bool

	// Execution tracking state
	instructionHistory []string
	progEnded          bool
}

func InitialModel() model {
	gp := make(map[arm64asm.Reg]uint64, 48)
	for i := arm64asm.X0; i <= arm64asm.X30; i++ {
		gp[i] = 0
	}

	ram := make(map[uint64]byte, 0)
	// 2k of ram
	for i := uint64(0x0000000000000000); i < 0x00000000000007FF; i++ {
		ram[i] = byte(0x00)
	}

	return model{
		gpRegister:         gp,
		mem:                ram,
		pc:                 0x0000000000000000,
		lastMemWrite:       false,
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
		rawProgBytes, err := GetTextSectionBytes("asm/demo")
		if err != nil {
			log.Fatalf("could not load binary into memory %v", err)
		}
		m.loadProgram(rawProgBytes)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			if !m.progEnded {
				err := m.step()
				if err != nil {
					return m, nil
				}
			}
		}
	}
	return m, nil
}

func (m *model) loadProgram(prog []byte) {
	for i := uint64(0); i < uint64(len(prog)); i += 4 {
		if i+3 < uint64(len(prog)) {
			m.mem[i] = prog[i]
			m.mem[i+1] = prog[i+1]
			m.mem[i+2] = prog[i+2]
			m.mem[i+3] = prog[i+3]
		}
	}
}

func (m *model) step() error {
	inst, err := m.decode()
	if err != nil {
		return err
	}

	m.instructionHistory = append(m.instructionHistory, inst.String())

	err = m.execute(inst)
	if err != nil {
		m.progEnded = true
		return err
	}

	m.pc += 4
	return nil
}

func (m *model) writeMemory(addr uint64, val byte) {
	m.mem[addr] = val
}

func (m *model) decode() (arm64asm.Inst, error) {
	instBytes := make([]byte, 4)
	instBytes[0] = m.mem[uint64(m.pc)]
	instBytes[1] = m.mem[uint64(m.pc+1)]
	instBytes[2] = m.mem[uint64(m.pc+2)]
	instBytes[3] = m.mem[uint64(m.pc+3)]

	decoded, err := arm64asm.Decode(instBytes)
	if err != nil {
		return decoded, err
	}
	return decoded, nil
}

/*
* use this for inacessible fields that should be exposed
* https://github.com/golang/go/issues/51517
 */
func regExtnToReg(arg arm64asm.RegExtshiftAmount) (arm64asm.Reg, error) {
	argValue := reflect.ValueOf(arg)
	regField := argValue.FieldByName("reg")

	if !regField.IsValid() {
		return 0, fmt.Errorf("reg field is not valid")
	}

	regInt := regField.Uint()
	return arm64asm.Reg(regInt), nil
}

// 32 bit reg not supported
func (m *model) execute(inst arm64asm.Inst) error {
	switch inst.Op {
	case arm64asm.ADD:
		destReg, drock := inst.Args[0].(arm64asm.Reg)
		sourceReg, srok := inst.Args[1].(arm64asm.Reg)
		if !drock || !srok {
			return fmt.Errorf("ADD invalid source or destination register %s", inst.String())
		}

		// source + immediate
		imm, ok := inst.Args[2].(arm64asm.Imm64)
		if ok {
			m.gpRegister[destReg] = m.gpRegister[sourceReg] + imm.Imm
			return nil
		}

		// dual register
		addReg, arock := inst.Args[2].(arm64asm.RegExtshiftAmount)
		if arock {
			hackReg, err := regExtnToReg(addReg)
			if err != nil {
				return err
			}
			m.gpRegister[destReg] = m.gpRegister[sourceReg] + m.gpRegister[hackReg]
			return nil
		} else {
			return fmt.Errorf("ADD invalid third ARG %s: %t", inst.String(), inst.Args[2])
		}
	case arm64asm.MOV:
		reg, rok := inst.Args[0].(arm64asm.Reg)
		imm, ok := inst.Args[1].(arm64asm.Imm64)

		if !rok {
			return fmt.Errorf("MOV invalid destination register %s", inst.String())
		}

		// register + register
		if ok {
			m.gpRegister[reg] = imm.Imm
			return nil
		}

		secondReg, srock := inst.Args[1].(arm64asm.Reg)
		if srock {
			m.gpRegister[reg] = m.gpRegister[secondReg]
			return nil
		} else {
			return fmt.Errorf("MOV invalid third ARG %s", inst.String())
		}

	case arm64asm.SUB:
		destReg, drock := inst.Args[0].(arm64asm.Reg)
		sourceReg, srok := inst.Args[1].(arm64asm.Reg)
		if !drock || !srok {
			return fmt.Errorf("SUB invalid source or destination register %s", inst.String())
		}

		imm, ok := inst.Args[2].(arm64asm.Imm64)
		if ok {
			m.gpRegister[destReg] = m.gpRegister[sourceReg] - imm.Imm
			return nil
		}

		addReg, arock := inst.Args[2].(arm64asm.RegExtshiftAmount)
		if arock {
			hackReg, err := regExtnToReg(addReg)
			if err != nil {
				return err
			}
			m.gpRegister[destReg] = m.gpRegister[sourceReg] - m.gpRegister[hackReg]
			return nil
		} else {
			return fmt.Errorf("SUB invalid third ARG %s: %t", inst.String(), inst.Args[2])
		}
	case arm64asm.SVC:
		if m.gpRegister[arm64asm.X16] == 1 {
			return errors.New("exit syscall invoked")
		} else {
			fmt.Print("unknown syscall\n")
		}
	case arm64asm.STR:
		valReg, valRok := inst.Args[0].(arm64asm.Reg)

		if !valRok {
			return fmt.Errorf("STR invalid base register %s", inst.String())
		}

		// base register addressing mode first reg has value second reg has memory address
		// check if there are only two args and they are both valid register
		srcReg, srcRock := inst.Args[1].(arm64asm.Reg)

		if srcRock && len(inst.Args) == 2 {
			addr := m.gpRegister[srcReg]
			val := m.gpRegister[valReg]
			for i := 0; i < 8; i++ {
				m.writeMemory(addr+uint64(i), byte(val>>(i*8)))
			}
			return nil
		}

		// base + immediate offset
		imm, ok := inst.Args[2].(arm64asm.Imm64)
		if srcRock && ok {
			addr := m.gpRegister[srcReg]
			offset := uint64(int64(imm.Imm))
			fAddr := addr + offset
			val := m.gpRegister[valReg]

			for i := 0; i < 8; i++ {
				m.writeMemory(fAddr+uint64(i), byte(val>>(i*8)))
			}
			return nil
		}

		// base + register offset
		offReg, offRock := inst.Args[2].(arm64asm.RegExtshiftAmount)
		if offRock {
			addr := m.gpRegister[srcReg]
			hackReg, err := regExtnToReg(offReg)
			if err != nil {
				return err
			}
			offset := uint64(int64(m.gpRegister[hackReg]))
			fAddr := addr + offset
			val := m.gpRegister[valReg]

			for i := 0; i < 8; i++ {
				m.writeMemory(fAddr+uint64(i), byte(val>>(i*8)))
			}
			return nil
		} else {
			return fmt.Errorf("invalid str %s", inst.String())
		}
	default:
		return fmt.Errorf("unknown instruction %s", inst.String())
	}
	return nil
}

func (m model) buildMemoryState() string {
	var stateBuilder strings.Builder
	stateBuilder.WriteString("   address \t\t    dec \t\t hex  \t\t ASCII\n")

	// show a window of 30 addresses with byte values
	if !m.lastMemWrite { // if memory was not altered in instruction show section of memory with instructions
		// if address is less than <=30
		if m.pc-15 <= 0 {
			for i := range 30 {
				if i == m.pc {
					fmt.Fprintf(&stateBuilder, "-> 0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", i, m.mem[uint64(i)], m.mem[uint64(i)], byteToAscii(m.mem[uint64(i)]))
				} else {
					fmt.Fprintf(&stateBuilder, "   0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", i, m.mem[uint64(i)], m.mem[uint64(i)], byteToAscii(m.mem[uint64(i)]))
				}
			}
		} else { // build a window
			bottom := m.pc - 15
			for i := range 30 {
				if bottom+i == m.pc {
					fmt.Fprintf(&stateBuilder, "-> 0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", bottom+i, m.mem[uint64(bottom+i)], m.mem[uint64(bottom+i)], byteToAscii(m.mem[uint64(bottom+i)]))
				} else {
					fmt.Fprintf(&stateBuilder, "   0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", bottom+i, m.mem[uint64(bottom+i)], m.mem[uint64(bottom+i)], byteToAscii(m.mem[uint64(bottom+i)]))
				}
			}
		}
	} else {
		// place last modified address in middle of window
	}

	return stateBuilder.String()
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

func (m model) buildGPRegisterstate() string {
	var stateBuilder strings.Builder
	for i := arm64asm.X0; i <= arm64asm.X30; i++ {
		fmt.Fprintf(&stateBuilder, "%s: %d\n", i.String(), m.gpRegister[i])
	}
	// format pc with padded hex. I think this is more intuitive to look for address where reg value is more intuitive with decimal
	fmt.Fprintf(&stateBuilder, "pc: 0x%016x", m.pc)
	return stateBuilder.String()
}

func byteToAscii(value byte) string {
	if value >= 32 && value <= 126 {
		return string(value)
	} else {
		return "."
	}
}

func (m model) View() string {
	titleContent := titleStyle.
		Foreground(lipgloss.Color("20")).
		Align(lipgloss.Left).
		Height(1).
		Render("ARMITUI - intuitive ARM stepper")

	regContent := titleStyle.Render("Registers") + "\n" + boxStyle.Render(m.buildGPRegisterstate())
	instContent := titleStyle.Render("Instruction History") + "\n" + boxStyle.Render(m.buildInstructionHistory())
	memContent := titleStyle.Render("Memory") + "\n" + boxStyle.Width(70).Render(m.buildMemoryState())

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
