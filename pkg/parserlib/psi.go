package parserlib

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
