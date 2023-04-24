package parserlib

import (
	"fmt"
)

type ParseError struct {
	msg      string
	pos      Position
	innerErr *ParseError
	input    string
}

func (pe *ParseError) Error() string {
	if pe.innerErr != nil {
		return fmt.Sprintf("%s: %s: %s", pe.pos.CompactString(), pe.msg, pe.innerErr)
	}
	return fmt.Sprintf("%s: %s", pe.pos.CompactString(), pe.msg)
}

func (pe *ParseError) InnermostError() *ParseError {
	if pe.innerErr == nil {
		return pe
	}
	return pe.innerErr.InnermostError()
}

func (pe *ParseError) ShowInContext() string {
	innermost := pe.InnermostError()
	return fmt.Sprintf(
		"%s: %s\n%s", innermost.pos.String(), innermost.msg, innermost.pos.ShowInContext(pe.input),
	)
}
