package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
)

func (s *schema) planSelect(query *Select) (lang.Expr, error) {
	tableDesc, ok := s.tables[query.Table]
	if !ok {
		return nil, fmt.Errorf("no such table: %s", query.Table)
	}

	// Scan the table.
	var innermostExpr lang.Expr
	innermostExpr = lang.NewMemberAccess(
		lang.NewMemberAccess(
			lang.NewMemberAccess(
				lang.NewVar("tables"),
				query.Table,
			),
			tableDesc.primaryKey,
		),
		"scan",
	)

	if query.Where != nil {
		innermostExpr = lang.NewFuncCall("filter", []lang.Expr{
			innermostExpr,
			lang.NewELambda(
				[]lang.Param{
					{
						Name: "row",
						Typ:  tableDesc.getType(),
					},
				},
				// TODO: use intEq if it's not a string...
				lang.NewFuncCall("strEq", []lang.Expr{
					lang.NewMemberAccess(lang.NewVar("row"), query.Where.ColumnName),
					lang.NewStringLit(query.Where.Value),
				}),
				lang.TBool,
			),
		})
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
		innermostExpr,
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
