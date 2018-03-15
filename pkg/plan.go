package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
)

func (s *schema) planSelect(query *Select) (lang.Expr, error) {
	if query.Where != nil {
		return nil, fmt.Errorf("don't know how to plan queries with WHERE yet")
	}

	tableDesc, ok := s.tables[query.Table]
	if !ok {
		return nil, fmt.Errorf("no such table: %s", query.Table)
	}

	// Get types and expressions for selectinos.
	types := map[string]lang.Type{}
	exprs := map[string]lang.Expr{}

	for _, selection := range query.Selections {
		if selection.SubSelect != nil {
			return nil, fmt.Errorf("don't know how to plan selects with subselects yet")
		}

		colDesc, err := tableDesc.getColDesc(selection.Name)
		if err != nil {
			return nil, fmt.Errorf("no such column: %s.%s", query.Table, selection.Name)
		}

		types[selection.Name] = colDesc.typ
		exprs[selection.Name] = lang.NewMemberAccess(lang.NewVar("row"), selection.Name)
	}

	// Build expression: a scan on the primary key.
	return lang.NewFuncCall("map", []lang.Expr{
		lang.NewMemberAccess(
			lang.NewMemberAccess(
				lang.NewVar(query.Table),
				tableDesc.primaryKey,
			),
			"scan",
		),
		lang.NewELambda(
			[]lang.Param{
				{
					Typ:  tableDesc.getType(),
					Name: "row",
				},
			},
			lang.NewRecordLit(exprs),
			lang.NewRecordType(types),
		),
	}), nil
}
