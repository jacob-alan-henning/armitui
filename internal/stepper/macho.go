package stepper

import (
	"debug/macho"
	"fmt"
)

func GetTextSectionBytes(filename string) ([]byte, error) {
	file, err := macho.Open(filename)
	if err != nil {
		return make([]byte, 0), err
	}
	defer file.Close()

	var textSection *macho.Section
	for _, section := range file.Sections {
		// find the executable machine code
		if section.Name == "__text" && section.Seg == "__TEXT" {
			textSection = section
			break
		}
	}

	if textSection == nil {
		return make([]byte, 0), fmt.Errorf("could not find __text section")
	}

	data, err := textSection.Data()
	if err != nil {
		return make([]byte, 0), err
	}

	return data, nil
}
