package arm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/arch/arm64/arm64asm"
)

type ARM64CPU struct {
	gpRegister   map[arm64asm.Reg]uint64
	mem          map[uint64]byte
	pc           int
	lastMemWrite bool // flag to check if the current instruction has modified memory
}

func (cpu ARM64CPU) BuildMemoryState() string {
	var stateBuilder strings.Builder
	stateBuilder.WriteString("   address \t\t    dec \t\t hex  \t\t ASCII\n")

	// show a window of 30 addresses with byte values

	if !cpu.lastMemWrite { // if memory was not altered in instruction show section of memory with instructions

		// if address is less than <=30
		if cpu.pc-15 <= 0 {
			for i := range 30 {
				if i == cpu.pc {
					fmt.Fprintf(&stateBuilder, "-> 0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", i, cpu.mem[uint64(i)], cpu.mem[uint64(i)], byteToAscii(cpu.mem[uint64(i)]))
				} else {
					fmt.Fprintf(&stateBuilder, "   0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", i, cpu.mem[uint64(i)], cpu.mem[uint64(i)], byteToAscii(cpu.mem[uint64(i)]))
				}
			}
		} else { // build a window
			bottom := cpu.pc - 15
			for i := range 30 {
				if bottom+i == cpu.pc {
					fmt.Fprintf(&stateBuilder, "-> 0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", bottom+i, cpu.mem[uint64(bottom+i)], cpu.mem[uint64(bottom+i)], byteToAscii(cpu.mem[uint64(bottom+i)]))
				} else {
					fmt.Fprintf(&stateBuilder, "   0x%08x \t\t %3d \t\t 0x%02x \t\t %s\n", bottom+i, cpu.mem[uint64(bottom+i)], cpu.mem[uint64(bottom+i)], byteToAscii(cpu.mem[uint64(bottom+i)]))
				}
			}
		}
	} else {
		// place last modified address in middle of window
	}

	return stateBuilder.String()
}

func byteToAscii(value byte) string {
	if value >= 32 && value <= 126 {
		return string(value)
	} else {
		return "."
	}
}

func (cpu ARM64CPU) GetPC() int {
	return cpu.pc
}

func (cpu ARM64CPU) GetCurrentInst() string {
	currentInst, _ := cpu.decode()
	return currentInst.String()
}

func (cpu ARM64CPU) BuildProgramState() string {
	var stateBuilder strings.Builder

	currentInst, _ := cpu.decode()

	fmt.Fprintf(&stateBuilder, "\nCURRENT:\n→ 0x%08x: %s\n", cpu.pc, currentInst.String())

	return stateBuilder.String()
}

func (cpu ARM64CPU) BuildGPRegisterstate() string {
	var stateBuilder strings.Builder
	for i := arm64asm.X0; i <= arm64asm.X30; i++ {
		fmt.Fprintf(&stateBuilder, "%s: %d\n", i.String(), cpu.gpRegister[i])
	}
	// format pc with padded hex. I think this is more intuitive to look for address where reg value is more intuitive with decimal
	fmt.Fprintf(&stateBuilder, "pc: 0x%016x", cpu.pc)
	return stateBuilder.String()
}

func (cpu ARM64CPU) PrintGPRegisterstate() {
	for i := arm64asm.X0; i <= arm64asm.X30; i++ {
		fmt.Printf("%s: %d\n", i.String(), cpu.gpRegister[i])
	}
}

func (cpu ARM64CPU) writeMemory(addr uint64, val byte) {
	cpu.mem[addr] = val
}

func (cpu ARM64CPU) LoadProgram(prog []byte) {
	for i := uint64(0); i < uint64(len(prog)); i += 4 {
		if i+3 < uint64(len(prog)) {
			cpu.mem[i] = prog[i]
			cpu.mem[i+1] = prog[i+1]
			cpu.mem[i+2] = prog[i+2]
			cpu.mem[i+3] = prog[i+3]
		}
	}
}

func NewArm64CPU() ARM64CPU {
	gp := make(map[arm64asm.Reg]uint64, 48)

	for i := arm64asm.X0; i <= arm64asm.X30; i++ {
		gp[i] = 0
	}

	ram := make(map[uint64]byte, 0)

	// 2k of ram
	for i := uint64(0x0000000000000000); i < 0x00000000000007FF; i++ {
		ram[i] = byte(0x00)
	}

	return ARM64CPU{
		gp,
		ram,
		0x0000000000000000,
		false,
	}
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
func (cpu *ARM64CPU) execute(inst arm64asm.Inst) error {
	switch inst.Op {
	case arm64asm.ADD:
		destReg, drock := inst.Args[0].(arm64asm.Reg)
		sourceReg, srok := inst.Args[1].(arm64asm.Reg)
		if !drock || !srok {
			return fmt.Errorf("ADD invalid source or destination register %s", inst.String())
		}

		imm, ok := inst.Args[2].(arm64asm.Imm64)
		if ok {
			cpu.gpRegister[destReg] = cpu.gpRegister[sourceReg] + imm.Imm
		} else {
			addReg, arock := inst.Args[2].(arm64asm.RegExtshiftAmount)
			if arock {
				hackReg, err := regExtnToReg(addReg)
				if err != nil {
					return err
				}
				cpu.gpRegister[destReg] = cpu.gpRegister[sourceReg] + cpu.gpRegister[hackReg]
			} else {
				return fmt.Errorf("ADD invalid third ARG %s: %t", inst.String(), inst.Args[2])
			}
		}
	case arm64asm.MOV:
		reg, rok := inst.Args[0].(arm64asm.Reg)
		imm, ok := inst.Args[1].(arm64asm.Imm64)

		if !rok {
			return fmt.Errorf("MOV invalid destination register %s", inst.String())
		}
		if ok {
			cpu.gpRegister[reg] = imm.Imm
		} else {
			secondReg, srock := inst.Args[1].(arm64asm.Reg)
			if srock {
				cpu.gpRegister[reg] = cpu.gpRegister[secondReg]
			} else {
				return fmt.Errorf("MOV invalid third ARG %s", inst.String())
			}
		}
	case arm64asm.SUB:
		destReg, drock := inst.Args[0].(arm64asm.Reg)
		sourceReg, srok := inst.Args[1].(arm64asm.Reg)
		if !drock || !srok {
			return fmt.Errorf("SUB invalid source or destination register %s", inst.String())
		}

		imm, ok := inst.Args[2].(arm64asm.Imm64)
		if ok {
			cpu.gpRegister[destReg] = cpu.gpRegister[sourceReg] - imm.Imm
		} else {
			addReg, arock := inst.Args[2].(arm64asm.RegExtshiftAmount)
			if arock {
				hackReg, err := regExtnToReg(addReg)
				if err != nil {
					return err
				}
				cpu.gpRegister[destReg] = cpu.gpRegister[sourceReg] - cpu.gpRegister[hackReg]
			} else {
				return fmt.Errorf("SUB invalid third ARG %s: %t", inst.String(), inst.Args[2])
			}
		}
	case arm64asm.SVC:
		if cpu.gpRegister[arm64asm.X16] == 1 {
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
			addr := cpu.gpRegister[srcReg]
			val := cpu.gpRegister[valReg]
			for i := 0; i < 8; i++ {
				cpu.writeMemory(addr+uint64(i), byte(val>>(i*8)))
			}
		}

		// base + immediate offset
		imm, ok := inst.Args[2].(arm64asm.Imm64)
		if srcRock && ok {
			addr := cpu.gpRegister[srcReg]
			offset := uint64(int64(imm.Imm))
			fAddr := addr + offset
			val := cpu.gpRegister[valReg]

			for i := 0; i < 8; i++ {
				cpu.writeMemory(fAddr+uint64(i), byte(val>>(i*8)))
			}
		}

		// base + register offset
		offReg, offRock := inst.Args[2].(arm64asm.RegExtshiftAmount)
		if srcRock && offRock {
			addr := cpu.gpRegister[srcReg]
			hackReg, err := regExtnToReg(offReg)
			if err != nil {
				return err
			}
			offset := uint64(int64(cpu.gpRegister[hackReg]))
			fAddr := addr + offset
			val := cpu.gpRegister[valReg]

			for i := 0; i < 8; i++ {
				cpu.writeMemory(fAddr+uint64(i), byte(val>>(i*8)))
			}
		}
	default:
		return fmt.Errorf("unknown instruction %s", inst.String())
	}
	return nil
}

func (cpu *ARM64CPU) decode() (arm64asm.Inst, error) {
	instBytes := make([]byte, 4)
	instBytes[0] = cpu.mem[uint64(cpu.pc)]
	instBytes[1] = cpu.mem[uint64(cpu.pc+1)]
	instBytes[2] = cpu.mem[uint64(cpu.pc+2)]
	instBytes[3] = cpu.mem[uint64(cpu.pc+3)]

	decoded, err := arm64asm.Decode(instBytes)
	if err != nil {
		return decoded, err
	}
	return decoded, nil
}

func (cpu *ARM64CPU) Step() error {
	inst, err := cpu.decode()
	if err != nil {
		return err
	}

	err = cpu.execute(inst)
	if err != nil {
		return err
	}
	cpu.pc += 4
	return nil
}
