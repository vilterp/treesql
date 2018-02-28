package parserlib

import "fmt"

type Position struct {
	Line   int
	Col    int
	Offset int
}

func (pos *Position) String() string {
	return fmt.Sprintf("line %d, col %d", pos.Line, pos.Col)
}

func (pos *Position) CompactString() string {
	return fmt.Sprintf("%d:%d", pos.Line, pos.Col)
}

func (pos *Position) MoreOnLine(n int) Position {
	return Position{
		Col:    pos.Col + n,
		Line:   pos.Line,
		Offset: pos.Offset + n,
	}
}

func (pos *Position) Newline() Position {
	return Position{
		Col:    1,
		Line:   pos.Line + 1,
		Offset: pos.Offset + 1,
	}
}
