package main

import (
	"fmt"
	"log"

	"github.com/jacob-alan-henning/armitui/internal/arm"
	"github.com/jacob-alan-henning/armitui/internal/macho"
)

func main() {
	rawProgBytes, err := macho.GetTextSectionBytes("asm/demo")
	if err != nil {
		log.Fatalf("could not load binary into memory %v", err)
	}

	cpu := arm.NewArm64CPU()
	cpu.LoadProgram(rawProgBytes)

	for {
		err := cpu.Step()
		if err != nil {
			fmt.Printf("program stopped execution %v\n", err)
			break
		}
	}
	cpu.PrintGPRegisterstate()
}
