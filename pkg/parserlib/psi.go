package parserlib

import (
	pp "github.com/vilterp/treesql/pkg/prettyprint"
)

type PSINode interface {
	GetChildren() []PSINode
	GetName() string
	GetSpan() SourceSpan
}

type Language struct {
	Grammar        *Grammar
	ParseTreeToPSI func(tt *TraceTree) PSINode
}

type SerializedPSINode struct {
	Children []*SerializedPSINode
	Name     string
	Span     SourceSpan
}

func SerializePSINode(n PSINode) *SerializedPSINode {
	children := n.GetChildren()
	serChildren := make([]*SerializedPSINode, len(children))
	for i, child := range children {
		serChildren[i] = SerializePSINode(child)
	}
	return &SerializedPSINode{
		Name:     n.GetName(),
		Span:     n.GetSpan(),
		Children: serChildren,
	}
}

func PrintPSINode(n PSINode) pp.Doc {
	children := n.GetChildren()
	childDocs := make([]pp.Doc, len(children))
	for i, child := range children {
		childDocs[i] = PrintPSINode(child)
	}

	return pp.Seq([]pp.Doc{
		pp.Text(n.GetName()),
		pp.Text("@"),
		pp.Text(n.GetSpan().From.CompactString()),
		pp.Text("-"),
		pp.Text(n.GetSpan().From.CompactString()),
		pp.Text("["),
		pp.Nest(2, pp.Join(childDocs, pp.CommaNewline)),
		pp.Text("]"),
	})
}
