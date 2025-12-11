package sqlparser

import (
	"fmt"
	"strings"
)

type Pragma struct {
	pragmas []string
}

func (d Pragma) PragmaIncludeIf() []string {
	return d.pragmas
}

func (d *Pragma) parseSinglePragma(s *Scanner) error {
	pragma := strings.TrimSpace(strings.TrimPrefix(s.Token(), "--sqlcode:"))
	if pragma == "" {
		return nil
	}
	parts := strings.Split(pragma, " ")

	if len(parts) != 2 || parts[0] != "include-if" {
		return fmt.Errorf("Illegal pragma: %s", s.Token())
	}

	d.pragmas = append(d.pragmas, strings.Split(parts[1], ",")...)
	return nil
}

func (d *Pragma) ParsePragmas(s *Scanner) error {
	for s.TokenType() == PragmaToken {
		err := d.parseSinglePragma(s)
		if err != nil {
			return err
		}
		s.NextNonWhitespaceToken()
	}

	return nil
}
