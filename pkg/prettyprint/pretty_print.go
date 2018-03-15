package prettyprint

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
	String() string
	// String returns a representation of the doc tree, for debugging.
	Debug() string
}

// Text

// tried to alias this to string; didn't work
type text struct {
	str string
}

var _ Doc = &text{}

func Text(s string) *text {
	return &text{
		str: s,
	}
}

func Textf(format string, args ...interface{}) *text {
	return Text(fmt.Sprintf(format, args...))
}

func (s *text) String() string {
	return s.str
}

func (s *text) Debug() string {
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

func (n *nest) String() string {
	indent := strings.Repeat(" ", n.nestBy)
	lines := strings.Split(n.doc.String(), "\n")
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

func (n *nest) Debug() string {
	return fmt.Sprintf("Nest(%d, %s)", n.nestBy, n.doc.String())
}

// Empty

type empty struct{}

var Empty = &empty{}

func (e *empty) String() string {
	return ""
}

func (empty) Debug() string {
	return "Empty"
}

// Seq

type concat struct {
	docs []Doc
}

func Seq(docs []Doc) *concat {
	return &concat{
		docs: docs,
	}
}

func (c *concat) String() string {
	buf := bytes.NewBufferString("")
	for _, doc := range c.docs {
		buf.WriteString(doc.String())
	}
	return buf.String()
}

func (c *concat) Debug() string {
	docStrs := make([]string, len(c.docs))
	for idx := range c.docs {
		docStrs[idx] = c.docs[idx].String()
	}
	return fmt.Sprintf("Seq(%s)", strings.Join(docStrs, ", "))
}

// Newline

type newline struct{}

var Newline = &newline{}

func (newline) String() string {
	return "\n"
}

func (newline) Debug() string {
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
	return Seq(out)
}

var Comma = Text(",")

var CommaNewline = Seq([]Doc{Comma, Newline})
