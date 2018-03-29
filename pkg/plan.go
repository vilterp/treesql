package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
)

func (s *schema) planSelect(query *Select, typeScope *lang.TypeScope) (lang.Expr, error) {
	return s.planSelectInternal(query, typeScope, 1)
}

func (s *schema) planSelectInternal(query *Select, typeScope *lang.TypeScope, depth int) (lang.Expr, error) {
	tableDesc, ok := s.tables[query.Table]
	if !ok {
		return nil, fmt.Errorf("no such table: %s", query.Table)
	}

	rowVarName := fmt.Sprintf("row%d", depth)

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
		if selection.SubSelect != nil {
			// Make the scan of the table we're joining to.
			innerRowVarName := fmt.Sprintf("row%d", depth+1)
			subSelectMap, err := s.planSelectInternal(selection.SubSelect, typeScope, depth+1)
			if err != nil {
				return nil, err
			}

			subSelectMapType, err := subSelectMap.GetType(typeScope)
			if err != nil {
				return nil, fmt.Errorf("error getting type of sub-select map: %s. expr: %s", err, subSelectMap.Format())
			}

			// Just doing the has many case first.
			// TODO: use an index instead of a filter
			otherTableDesc := s.tables[selection.SubSelect.Table]
			colReferencingThis := otherTableDesc.colReferencingTable(query.Table)
			if colReferencingThis == nil {
				return nil, fmt.Errorf(
					"couldn't find a column on %s referencing %s", otherTableDesc.name, query.Table,
				)
			}
			// Build filter
			filterLambdaBody := lang.NewFuncCall("strEq", []lang.Expr{
				lang.NewMemberAccess(lang.NewVar(innerRowVarName), *colReferencingThis),
				lang.NewMemberAccess(lang.NewVar(rowVarName), tableDesc.primaryKey),
			})

			subSelectFilter := lang.NewFuncCall("filter", []lang.Expr{
				subSelectMap,
				lang.NewELambda(
					[]lang.Param{
						{
							Name: innerRowVarName,
							Typ:  subSelectMapType.(*lang.TIterator).InnerType,
						},
					},
					filterLambdaBody,
					lang.TBool,
				),
			})

			newTS := lang.NewTypeScope(typeScope)
			newTS.Add(innerRowVarName, subSelectMapType.(*lang.TIterator).InnerType)

			fmt.Println("made a new scope:", newTS.Format())

			subSelectFilterType, err := subSelectFilter.GetType(newTS)
			if err != nil {
				return nil, fmt.Errorf(`error getting type of sub-select "%s": %v. expr: %s`, selection.Name, err, subSelectFilter.Format())
			}
			types[selection.Name] = subSelectFilterType
			exprs[selection.Name] = subSelectFilter
			continue
		}

		colDesc, err := tableDesc.getColDesc(selection.Name)
		if err != nil {
			return nil, fmt.Errorf("no such column: %s.%s", query.Table, selection.Name)
		}

		types[selection.Name] = colDesc.typ
		exprs[selection.Name] = lang.NewMemberAccess(lang.NewVar(rowVarName), selection.Name)
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
