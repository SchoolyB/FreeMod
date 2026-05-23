package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/freemod/freemod/trainers"
)

type ctEntry struct {
	XMLName      xml.Name   `xml:"CheatEntry"`
	Description  string     `xml:"Description"`
	VariableType string     `xml:"VariableType"`
	Address      string     `xml:"Address"`
	Value        string     `xml:"Value"`
	Children     []ctEntry  `xml:"CheatEntries>CheatEntry"`
}

type ctTable struct {
	XMLName xml.Name  `xml:"CheatTable"`
	Entries []ctEntry `xml:"CheatEntries>CheatEntry"`
}

func mapCTType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "8 bytes", "qword":
		return "int64"
	case "float":
		return "float32"
	case "double":
		return "float32"
	default:
		return "int32"
	}
}

var ctModuleRe = regexp.MustCompile(`"([^"]+)"\+([0-9A-Fa-f]+)`)

func parseCTAddress(addr string) (baseOffset string, offsets []string, exe string) {
	m := ctModuleRe.FindStringSubmatchIndex(addr)
	if m != nil {
		exe = addr[m[2]:m[3]]
		baseOffset = "0x" + addr[m[4]:m[5]]
		rest := addr[m[1]:]
		rest = strings.TrimLeft(rest, ", ")
		for _, part := range strings.FieldsFunc(rest, func(r rune) bool {
			return r == ',' || r == '+' || r == '[' || r == ']' || r == ' '
		}) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			part = strings.TrimPrefix(part, "0x")
			part = strings.TrimPrefix(part, "0X")
			_, err := strconv.ParseUint(part, 16, 64)
			if err == nil {
				offsets = append(offsets, "0x"+part)
			}
		}
		return
	}
	// No module match — try plain hex address
	clean := strings.TrimSpace(addr)
	clean = strings.Trim(clean, `"`)
	clean = strings.TrimPrefix(clean, "0x")
	clean = strings.TrimPrefix(clean, "0X")
	_, err := strconv.ParseUint(clean, 16, 64)
	if err == nil {
		baseOffset = "0x" + clean
	}
	return
}

func collectCTEntries(entries []ctEntry) []trainers.Cheat {
	var out []trainers.Cheat
	for _, e := range entries {
		if len(e.Children) > 0 {
			out = append(out, collectCTEntries(e.Children)...)
			continue
		}
		vt := strings.ToLower(strings.TrimSpace(e.VariableType))
		if vt == "" || vt == "auto assembler script" || strings.TrimSpace(e.Address) == "" {
			continue
		}
		base, offs, _ := parseCTAddress(e.Address)
		if base == "" {
			continue
		}
		var val float64
		if v, err := strconv.ParseFloat(strings.TrimSpace(e.Value), 64); err == nil {
			val = v
		}
		desc := strings.TrimSpace(e.Description)
		desc = strings.Trim(desc, `"`)
		if desc == "" {
			desc = "Cheat"
		}
		out = append(out, trainers.Cheat{
			Name:       desc,
			Type:       mapCTType(e.VariableType),
			Behavior:   "freeze",
			BaseOffset: base,
			Offsets:    offs,
			Value:      val,
			Input:      true,
		})
	}
	return out
}

// ParseCT parses Cheat Engine XML and returns a FreeMod trainer JSON string.
func (a *App) ParseCT(xmlContent string) (string, error) {
	var table ctTable
	if err := xml.Unmarshal([]byte(xmlContent), &table); err != nil {
		return "", fmt.Errorf("invalid CT XML: %w", err)
	}
	cheats := collectCTEntries(table.Entries)
	if len(cheats) == 0 {
		return "", fmt.Errorf("no compatible cheats found — CT file may use Auto Assembler Scripts only")
	}
	tf := trainers.TrainerFile{
		Game:    "",
		Exe:     "",
		Version: "1.0",
		Cheats:  cheats,
	}
	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
