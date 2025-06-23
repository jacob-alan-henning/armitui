package main

import (
	"fmt"
	"log"

	"github.com/jacob-alan-henning/armitui/internal/macho"
	"golang.org/x/arch/arm64/arm64asm"
)

func Fain() {
	machoTextSection, err := macho.GetTextSection("asm/demo")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("text section address 0x%x\n", machoTextSection.Address)

	data := machoTextSection.Raw
	maxInstructions := 50
	if len(data)/4 < maxInstructions {
		maxInstructions = len(data) / 4
	}

	fmt.Println("\nDisassembled instructions:")
	fmt.Println("ADDRESS       OP  ARGS")
	fmt.Println("------------------------------------------------------")

	for i := 0; i < maxInstructions; i++ {
		offset := i * 4
		addr := machoTextSection.Address + uint64(offset)

		bytes := data[offset : offset+4]
		decoded, err := arm64asm.Decode(bytes)
		if err != nil {
			fmt.Printf("0x%x:  ERROR: %v\n", addr, err)
			continue
		}

		// Print the address, machine code, and decoded instruction
		fmt.Printf("0x%x:  %s %s\n", addr, decoded.Op, argString(decoded.Args))
	}

	cpu := newARMCPU()
	prog := NewProgram(data, 50, machoTextSection.Address)

	for _, in := range prog.instructions {
		if in.instruction.Op.String() == "ADD" {
			cpu.X0 = 2
		}
		cpu.printRegState()
	}
}

type armCPU struct {
	X0 int64
	X1 int64
	X2 int64
	X3 int64
}

func newARMCPU() armCPU {
	return armCPU{
		0,
		0,
		0,
		0,
	}
}

func (cpu *armCPU) printRegState() {
	fmt.Println(cpu.X0)
}

func NewProgram(data []byte, maxInstructions int, address uint64) Program {
	if len(data)/4 < maxInstructions {
		maxInstructions = len(data) / 4
	}

	instrctions := make([]armInst, 0)

	for i := 0; i < maxInstructions; i++ {
		offset := i * 4
		addr := address + uint64(offset)

		bytes := data[offset : offset+4]
		decoded, err := arm64asm.Decode(bytes)
		if err != nil {
			fmt.Printf("0x%x:  ERROR: %v\n", addr, err)
			continue
		}

		newInst := armInst{
			instruction: decoded,
			address:     addr,
		}
		instrctions = append(instrctions, newInst)
	}
	return Program{
		instrctions,
	}
}

type validArmInst interface {
	execute(cpu *armCPU)
}

type armInst struct {
	instruction arm64asm.Inst
	address     uint64
}

type Program struct {
	instructions []armInst
}

func argString(args arm64asm.Args) string {
	val := ""
	for i := range args {
		if args[i] != nil {
			val = val + args[i].String() + " "
		}
	}
	return val
}
