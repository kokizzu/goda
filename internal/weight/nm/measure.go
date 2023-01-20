package nm

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Sym struct {
	Addr uint64
	Size int64
	Code rune // nm code (T for text, D for data, and so on)

	QualifiedName string
	Info          string

	Path []string
	Name string
}

func ParseBinary(binary string) ([]*Sym, error) {
	command := exec.Command("go", "tool", "nm", "-size", binary)

	reader, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("failed to start: %w", err)
	}

	var syms []*Sym
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		sym, err := parseLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("failed to parse: %w", err)
		}
		if sym.QualifiedName == "" {
			continue
		}

		if len(sym.Path) > 0 && strings.HasPrefix(sym.Path[0], "go.itab.") {
			continue
		}
		if len(sym.Path) > 0 && strings.HasPrefix(sym.Path[0], "type..") {
			continue
		}

		syms = append(syms, sym)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning failed: %w", err)
	}

	return syms, nil
}

func parseLine(s string) (*Sym, error) {
	var err error
	sym := &Sym{}

	tokens := strings.Fields(s)
	if len(tokens) <= 2 {
		return nil, fmt.Errorf("invalid sym text: %q", s)
	}

	addrField := ""
	sizeField := ""
	typeField := ""
	nameField := ""
	infoField := ""

	isSymType := func(s string) bool {
		return len(s) == 1 && unicode.IsLetter(rune(s[0]))
	}

	switch {
	case isSymType(tokens[1]):
		// in some cases addr is not printed
		sizeField = tokens[0]
		typeField = tokens[1]
		if len(tokens) > 2 {
			nameField = tokens[2]
		}
		if len(tokens) > 3 {
			infoField = strings.Join(tokens[3:], " ")
		}
	case isSymType(tokens[2]):
		addrField = tokens[0]
		sizeField = tokens[1]
		typeField = tokens[2]
		if len(tokens) > 3 {
			nameField = tokens[3]
		}
		if len(tokens) > 4 {
			infoField = strings.Join(tokens[4:], " ")
		}
	default:
		return nil, fmt.Errorf("unable to find type in sym: %q", s)
	}

	if addrField != "" {
		sym.Addr, err = strconv.ParseUint(addrField, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid addr: %q", addrField)
		}
	}

	if sizeField != "" {
		sym.Size, err = strconv.ParseInt(sizeField, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %q", s, sizeField)
		}

		// ignore external sym size
		if sym.Size == 4294967296 {
			sym.Size = 0
		}
	}

	if code := strings.TrimSpace(typeField); code != "" {
		sym.Code, _ = utf8.DecodeRuneInString(code)
	}

	sym.QualifiedName = nameField
	sym.Info = infoField

	if sym.QualifiedName == "" {
		return sym, nil
	}

	braceOff := strings.IndexByte(sym.QualifiedName, '(')
	if braceOff < 0 {
		braceOff = len(sym.QualifiedName)
	}

	slashPos := strings.LastIndexByte(sym.QualifiedName[:braceOff], '/')
	if slashPos < 0 {
		slashPos = 0
	}

	pointOff := strings.IndexByte(sym.QualifiedName[slashPos:braceOff], '.')
	if pointOff < 0 {
		pointOff = 0
	}

	p := slashPos + pointOff
	if p > 0 {
		sym.Path = strings.Split(sym.QualifiedName[:p], "/")
		sym.Name = sym.QualifiedName[p+1:]
	} else {
		sym.Name = sym.QualifiedName
	}

	return sym, nil
}
