// Copyright 2018-2019 "Misato's Angel" <misatos.arngel@gmail.com>.
// Use of this source code is governed the MIT license.
// license that can be found in the LICENSE file.
package gitconfig

import (
	"bufio"
	"fmt"
	"unicode"
)

type Parser struct {
	Reader           *bufio.Scanner
	Config           *Config
	lineNo           uint64
	charPos          uint64
	curLine          string
	hadNonWhiteSpace bool
	inEscape         bool
	inQuote          bool
	section          string
	subSection       string
}

// advance to the next line
func (self *Parser) ReadLine() bool {
	if !self.Reader.Scan() {
		return false
	}
	self.lineNo = self.lineNo + 1
	self.charPos = 0
	self.curLine = self.Reader.Text()
	return true
}

func (self *Parser) GetCurLine() string {
	if int(self.charPos) >= len(self.curLine) {
		return ""
	}
	return self.curLine[self.charPos:]
}

func (self *Parser) Read() error {
	for self.ReadLine() {
		if self.curLine == "" {
			continue
		}
		if err := self.readKeyOrSection(); err != nil {
			return err
		}
	}
	return nil
}

func (self *Parser) readKeyOrSection() error {
	hadNonWhiteSpace := false
	inQuote := false
	inEscape := false
	out := ""
	text := self.GetCurLine()
	for _, r := range text {
		self.charPos++
		if unicode.IsSpace(r) {
			if !hadNonWhiteSpace {
				continue
			}
			if inEscape {
				return self.makeError(fmt.Sprintf("Unrecognised escape character: '%s'", string(r)))
			}
			if inQuote {
				out += string(r)
			}
			continue
		}
		if r == ';' || r == '#' {
			return nil // dead line
		}
		hadNonWhiteSpace = true
		// backup the char again
		self.charPos--
		if r == '[' {
			return self.readSection()
		}
		return self.readKeyValue()
	}
	return nil
}

func (self *Parser) readSection() error {
	inSection := false
	self.section = ""
	self.subSection = ""
	text := self.GetCurLine()
	for _, r := range text {
		self.charPos++
		if unicode.IsSpace(r) {
			continue
		}
		if r == ';' || r == '#' {
			if inSection {
				return self.makeError(fmt.Sprintf("Unexpected %s in section name '%s", string(r), self.section))
			}
			return nil // comments the line
		}
		if r == '[' {
			if self.section != "" || inSection {
				return self.makeError(fmt.Sprintf("Unexpected [ in section name '%s'", self.section))
			}
			inSection = true
			continue
		}
		if r == ']' {
			if !inSection {
				return self.makeError(fmt.Sprintf("Unexpected ] in section name '%s'", self.section))
			}
			// section declarations may be immediately followed by key = value on the same line
			return self.readKeyValue()
		}
		if r == '"' {
			if self.subSection == "" {
				if self.section == "" {
					return self.makeError(fmt.Sprintf("Unexpected \" before section name"))
				}
				self.charPos--
				return self.readSubsection()
			}
			return self.makeError(fmt.Sprintf("Unexpected \" in section name '%s'", self.section))
		}
		self.section += string(r)
	}
	return self.makeError(fmt.Sprintf("Unexpected end of line when reading section"))
}

// looks for a quoted string inside a section name e.g. "foo" from [bar "foo"]
func (self *Parser) readSubsection() error {
	inSubSection := false
	inEscape := false
	self.subSection = ""
	text := self.GetCurLine()
	for _, r := range text {
		self.charPos++
		if inEscape {
			inEscape = false
			switch r {
			case '"':
				self.subSection += "\""
			case 't':
				self.subSection += "\t"
			case 'n':
				self.subSection += "\n"
			case '\\':
				self.subSection += "\\"
			}
			continue // all escaped chars get lost in subsection names...
		}
		if unicode.IsSpace(r) {
			if inSubSection {
				self.subSection += string(r)
			}
			continue
		}
		if r == '"' {
			if inSubSection {
				return nil
			}
			inSubSection = true
			continue
		}
		if r == '\\' {
			inEscape = true
			continue
		}
		self.subSection += string(r)
	}
	return self.makeError(fmt.Sprintf("Unexpected end of line when reading subsection"))
}

func (self *Parser) readKeyValue() error {
	hadNonWhiteSpace := false
	doneKey := false
	text := self.GetCurLine()
	key := ""
	for _, r := range text {
		self.charPos++
		if unicode.IsSpace(r) {
			if hadNonWhiteSpace {
				doneKey = true
			}
			continue
		}
		if r == '=' {
			value, err := self.readValue(false, "")
			if err != nil {
				return err
			}
			self.Config.AddKeyValue(self.section, self.subSection, key, &value)
			return nil
		}
		if doneKey {
			return self.makeError(fmt.Sprintf("Unexpected '%s' after key '%s', expected =, whitespace or newline\n", string(r), key))
		}
		// config keys must start with an ascii letter, after that they can contain '-' and digits too
		if !unicode.IsLetter(r) {
			if !hadNonWhiteSpace {
				return self.makeError(fmt.Sprintf("Unexpected '%s' starting key, expected a letter\n", string(r)))
			} else if r != '-' && !unicode.IsDigit(r) {
				return self.makeError(fmt.Sprintf("Unexpected '%s' in key, expected a ascii letter, hyphen or digit\n", string(r)))
			}
		}
		hadNonWhiteSpace = true
		key += string(r)
	}
	if key != "" {
		self.Config.AddKeyValue(self.section, self.subSection, key, nil)
	}
	return nil
}

func (self *Parser) readValue(hadNonWhiteSpace bool, spaceRun string) (string, error) {
	inEscape := false
	value := ""
	quoted := false
	text := self.GetCurLine()
	for _, r := range text {
		self.charPos++
		if unicode.IsSpace(r) {
			if hadNonWhiteSpace {
				spaceRun += string(r)
			}
			continue
		}
		if r == '\\' && !inEscape {
			inEscape = true
			continue
		}
		if !quoted && (r == ';' || r == '#') {
			// finish line?
			return value, nil
		}
		hadNonWhiteSpace = true
		if spaceRun != "" {
			// append any extra spaces
			value += spaceRun
			spaceRun = ""
		}
		// deal with line comment characters
		if r == ';' || r == '#' {
			value += string(r)
			continue
		}
		if inEscape {
			inEscape = false

			switch r {
			case '"':
				value += "\""
			case 't':
				value += "\t"
			case 'n':
				value += "\n"
			case '\\':
				value += "\\"
			default:
				return value, self.makeError(fmt.Sprintf("Unexpected '%s' in escape only double-quote, n, t and \\ are allowed to be escaped.\n", string(r)))
			}
			continue
		}
		if r == '"' {
			if quoted {
				quoted = false
			} else {
				quoted = true
			}
			continue
		}
		value += string(r)
	}
	if quoted {
		return value, self.makeError(fmt.Sprintf("Unexpected newline in quoted value string: '%s'.\n", value))
	}
	if inEscape {
		if self.ReadLine() {
			next, err := self.readValue(hadNonWhiteSpace, spaceRun)
			if err != nil {
				return value, err
			}
			return value + next, nil
		}
	}
	return value, nil
}

func (self *Parser) makeError(reason string) *ParseError {
	return &ParseError{
		Message: reason,
		Line:    self.curLine,
		LineNo:  self.lineNo,
		CharPos: self.charPos,
	}
}
