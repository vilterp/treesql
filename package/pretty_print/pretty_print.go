package pretty_print

import (
	"bytes"
	"fmt"
	"strings"
)

// Based on http://homepages.inf.ed.ac.uk/wadler/papers/prettier/prettier.pdf
// TODO: this is the paper's naive implementation; use the more efficient one
// TODO: alternative layout combinators (best, group, etc)

type Doc interface {
	// Render returns the pretty-printed representation.
	Render() string
	// String returns a representation of the doc tree, for debugging.
	String() string
}

// Text

// tried to alias this to string; didn't work
type text struct {
	str string
}

func Text(s string) *text {
	return &text{
		str: s,
	}
}

func Textf(format string, args ...interface{}) *text {
	return Text(fmt.Sprintf(format, args...))
}

func (s *text) Render() string {
	return s.str
}

func (s *text) String() string {
	return fmt.Sprintf("Text(%#v)", s.str)
}

// Nest

type nest struct {
	doc    Doc
	nestBy int
}

func Nest(by int, d Doc) Doc {
	return &nest{
		doc:    d,
		nestBy: by,
	}
}

func (n *nest) Render() string {
	indent := strings.Repeat(" ", n.nestBy)
	lines := strings.Split(n.doc.Render(), "\n")
	buf := bytes.NewBufferString("")
	for idx, line := range lines {
		if idx > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(indent)
		buf.WriteString(line)
	}
	return buf.String()
}

func (n *nest) String() string {
	return fmt.Sprintf("Nest(%d, %s)", n.nestBy, n.doc.String())
}

// Empty

type empty struct{}

var Empty = &empty{}

func (e *empty) Render() string {
	return ""
}

func (empty) String() string {
	return "Empty"
}

// Concat

type concat struct {
	docs []Doc
}

func Concat(docs []Doc) *concat {
	return &concat{
		docs: docs,
	}
}

func (c *concat) Render() string {
	buf := bytes.NewBufferString("")
	for _, doc := range c.docs {
		buf.WriteString(doc.Render())
	}
	return buf.String()
}

func (c *concat) String() string {
	docStrs := make([]string, len(c.docs))
	for idx := range c.docs {
		docStrs[idx] = c.docs[idx].String()
	}
	return fmt.Sprintf("Concat(%s)", strings.Join(docStrs, ", "))
}

// Newline

type newline struct{}

var Newline = &newline{}

func (newline) Render() string {
	return "\n"
}

func (newline) String() string {
	return "Newline"
}

// Combinators && stdlib

// TODO: make this split over line breaks if there's
// not enough width
// TODO: formatting with trailing commas would be nice
func Join(docs []Doc, sep Doc) Doc {
	var out []Doc
	for idx, doc := range docs {
		if idx > 0 {
			out = append(out, sep)
		}
		out = append(out, doc)
	}
	return Concat(out)
}

var CommaNewline = Concat([]Doc{Text(","), Newline})