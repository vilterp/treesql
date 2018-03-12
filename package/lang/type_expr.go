package lang

import (
	pp "github.com/vilterp/treesql/package/pretty_print"
)

type typeExpr interface {
	Format() pp.Doc

	Resolve(*TypeScope) (Type, error)
}

type normalType string

var _ typeExpr = NewNormalType("foo")

func NewNormalType(name string) *normalType {
	nt := normalType(name)
	return &nt
}

func (nt *normalType) Format() pp.Doc {
	return pp.Text(string(*nt))
}

func (nt *normalType) Resolve(ts *TypeScope) (Type, error) {
	return ts.find(string(*nt))
}

// TODO: parameterized type
