// Copyright 2018-2019 "Misato's Angel" <misatos.arngel@gmail.com>.
// Use of this source code is governed the MIT license.
// license that can be found in the LICENSE file.

package gitconfig

import (
	"fmt"
	"strings"
)

type ParseError struct {
	Message string
	Line    string
	LineNo  uint64
	CharPos uint64
}

func (self *ParseError) Error() string {
	out := fmt.Sprintf("Line: %d Char: %d\n%s\n", self.LineNo, self.CharPos, self.Line)
	if self.CharPos != 0 {
		out = out + strings.Repeat(" ", int(self.CharPos-1))
	}
	out = out + "^\n"
	return out + self.Message
}

type LoadError map[string]error

func (self LoadError) HaveErrors() bool {
	if cnt := len(self); cnt > 0 {
		return true
	}
	return false
}

func (self LoadError) Error() string {
	cnt := len(self)
	if cnt == 0 {
		return "No errors occurred"
	}
	if cnt == 1 {
		for k, v := range self {
			return fmt.Sprintf("When attempting to assign '%s':\n - %s\n", k, v)
		}
	}
	out := "The following errors occurred:\n"
	for k, v := range self {
		out += fmt.Sprintf("When attempting to assign '%s':\n - %s\n", k, v)
	}
	return out
}
