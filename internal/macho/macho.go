package macho

import (
	"debug/macho"
	"fmt"
)

type TextSection struct {
	Address uint64
	Raw     []byte
}

func newTextSection() TextSection {
	return TextSection{
		Address: 0,
		Raw:     make([]byte, 0),
	}
}

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

func GetTextSection(filename string) (TextSection, error) {
	text := newTextSection()

	file, err := macho.Open(filename)
	if err != nil {
		return text, err
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
		return text, fmt.Errorf("could not find __text section")
	}

	data, err := textSection.Data()
	if err != nil {
		return text, err
	}

	text.Raw = data
	text.Address = textSection.Addr

	return text, nil
}

type MachoHeader struct {
	MagicNumber uint32
	CpuType     uint32
	CpuSubType  uint32
	FileType    uint32
	Ncmds       uint32
	SizeOfCmds  uint32
	Flags       uint32
}

// load command types
const (
	LC_UUID = iota
	LC_SEGMENT
	LC_SEGMENT_64
	LC_SYMTAB
	LC_DYSMTAB
	LC_THREAD
	LC_UNIXTHREAD
	LC_LOAD_DYLIB
	LC_ID_DYLIB
	LC_PREBOUND_DYLIB
	LC_LOAD_DYLINKER
	LC_ID_DYLINKER
	LC_ROUTITNES
	LC_ROUTINES_64
	LC_TWOLEVEL_HINTS
	LC_SUBLEVEL_HINTS
	LC_SUB_FRAMEWORK
	LC_SUB_LIBRARY
	LC_SUB_CLIENT
)

type LoadCommand struct {
	Cmd     uint32
	CmdSize uint32
	Raw     []byte
}

type MachoFile struct {
	Header MachoHeader
}
