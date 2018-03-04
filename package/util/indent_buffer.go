package util

import (
	"bytes"
	"fmt"
	"strings"
)

type IndentBuffer struct {
	depth  int
	buf    *bytes.Buffer
	indent string
}

func NewIndentBuffer(indent string) *IndentBuffer {
	return &IndentBuffer{
		depth:  0,
		buf:    bytes.NewBufferString(""),
		indent: indent,
	}
}

func (ib *IndentBuffer) String() string {
	return ib.buf.String()
}

func (ib *IndentBuffer) Indent() {
	ib.depth++
}

func (ib *IndentBuffer) Dedent() {
	ib.depth--
}

//
//func (ib *IndentBuffer) Printf(format string, params ...interface{}) {
//	ib.buf.WriteString(strings.Repeat(ib.indent, ib.depth))
//	ib.buf.WriteString(fmt.Sprintf(format, params...))
//}

func (ib *IndentBuffer) Printlnf(format string, params ...interface{}) {
	ib.buf.WriteString(strings.Repeat(ib.indent, ib.depth))
	ib.buf.WriteString(fmt.Sprintf(format, params...))
	ib.buf.WriteByte('\n')
}
