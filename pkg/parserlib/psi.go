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
