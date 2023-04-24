package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
)

func (s *schema) planSelect(query *Select, typeScope *lang.TypeScope) (lang.Expr, error) {
	return s.planSelectInternal(query, typeScope, 1, nil)
}

func (s *schema) planSelectInternal(
	query *Select, typeScope *lang.TypeScope, depth int, filterLambda lang.Expr,
) (lang.Expr, error) {
	tableDesc, ok := s.tables[query.Table]
	if !ok {
		return nil, fmt.Errorf("no such table: %s", query.Table)
	}

	rowVarName := fmt.Sprintf("row%d", depth)

	// Scan the table.
	var innermostExpr lang.Expr
	innermostExpr = lang.NewFuncCall(
		"scan",
		[]lang.Expr{
			lang.NewMemberAccess(
				lang.NewMemberAccess(
					lang.NewVar("tables"),
					query.Table,
				),
				tableDesc.primaryKey,
			),
		},
	)

	if filterLambda != nil {
		innermostExpr = lang.NewFuncCall("filter", []lang.Expr{
			innermostExpr,
			filterLambda,
		})
	}

	if query.Where != nil {
		innermostExpr = lang.NewFuncCall("filter", []lang.Expr{
			innermostExpr,
			lang.NewELambda(
				[]lang.Param{
					{
						Name: rowVarName,
						Typ:  tableDesc.getType(),
					},
				},
				// TODO: use intEq if it's not a string...
				lang.NewFuncCall("strEq", []lang.Expr{
					lang.NewMemberAccess(lang.NewVar(rowVarName), query.Where.ColumnName),
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
		var selectionExpr lang.Expr
		var selectionType lang.Type
		if selection.SubSelect != nil {
			var err error
			selectionExpr, selectionType, err = s.getSubSelect(
				rowVarName, tableDesc, selection, typeScope, depth,
			)
			if err != nil {
				return nil, err
			}
		} else {
			colDesc, err := tableDesc.getColDesc(selection.Name)
			if err != nil {
				return nil, fmt.Errorf("no such column: %s.%s", query.Table, selection.Name)
			}
			selectionExpr = lang.NewMemberAccess(lang.NewVar(rowVarName), selection.Name)
			selectionType = colDesc.typ
		}

		exprs[selection.Name] = selectionExpr
		types[selection.Name] = selectionType
	}

	// Build expression: a scan on the primary key.
	return lang.NewFuncCall("map", []lang.Expr{
		innermostExpr,
		lang.NewELambda(
			[]lang.Param{
				{
					Typ:  tableDesc.getType(),
					Name: rowVarName,
				},
			},
			lang.NewRecordLit(exprs),
			lang.NewTRecord(types),
		),
	}), nil
}

func (s *schema) getSubSelect(
	rowVarName string, tableDesc *tableDescriptor, selection *Selection,
	typeScope *lang.TypeScope, depth int,
) (lang.Expr, lang.Type, error) {
	// Make the scan of the table we're joining to.
	innerRowVarName := fmt.Sprintf("row%d", depth+1)

	// Just doing the has many case first.
	// TODO: use an index instead of a filter
	innerTableDesc := s.tables[selection.SubSelect.Table]
	colReferencingThis := innerTableDesc.colReferencingTable(tableDesc.name)
	if colReferencingThis == nil {
		return nil, nil, fmt.Errorf(
			"couldn't find a column on %s referencing %s", innerTableDesc.name, tableDesc.name,
		)
	}
	// Build filter
	filterLambdaBody := lang.NewFuncCall("strEq", []lang.Expr{
		lang.NewMemberAccess(lang.NewVar(innerRowVarName), *colReferencingThis),
		lang.NewMemberAccess(lang.NewVar(rowVarName), tableDesc.primaryKey),
	})

	innerFilterLambda := lang.NewELambda(
		[]lang.Param{
			{
				Name: innerRowVarName,
				Typ:  innerTableDesc.getType(), // TODO: ???
			},
		},
		filterLambdaBody,
		lang.TBool,
	)

	innerTypeScope := typeScope.NewChildScope()
	innerTypeScope.Add(rowVarName, tableDesc.getType())

	subSelectExpr, err := s.planSelectInternal(
		selection.SubSelect, innerTypeScope, depth+1, innerFilterLambda,
	)
	if err != nil {
		return nil, nil, err
	}

	subSelectExprType, err := subSelectExpr.GetType(innerTypeScope)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"error getting type of sub-select: %s. expr:\n\n%s\n\nTS:\n\n%s",
			err, subSelectExpr.Format(), innerTypeScope.Format(),
		)
	}

	return subSelectExpr, subSelectExprType, nil
}
