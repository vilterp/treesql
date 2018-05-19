package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
)

type IndexMap map[string]map[string]*lang.VIndex

func (s *schema) planSelect(query *Select, typeScope *lang.TypeScope, indexMap IndexMap) (lang.Expr, lang.Type, error) {
	expr, err := s.planSelectInternal(query, typeScope, indexMap, 1, nil)
	if err != nil {
		return nil, nil, err
	}
	typ, err := expr.GetType(typeScope)
	if err != nil {
		return nil, nil, err
	}
	return expr, typ, err
}

func (s *schema) planSelectInternal(
	query *Select, typeScope *lang.TypeScope, indexMap IndexMap, depth int, join *oneToManyJoin,
) (lang.Expr, error) {
	tableDesc, ok := s.tables[query.Table]
	if !ok {
		return nil, fmt.Errorf("no such table: %s", query.Table)
	}

	rowVarName := fmt.Sprintf("row%d", depth)

	// Build expr for main collection we're scanning over.
	if query.Where != nil {
		return nil, fmt.Errorf("not supporting WHERE yet")
	}

	primaryIndexExpr := lang.NewEIndexRef(indexMap[query.Table][tableDesc.primaryKey])

	// Get types and expressions for selections.
	types := map[string]lang.Type{}
	exprs := map[string]lang.Expr{}
	for _, selection := range query.Selections {
		var selectionExpr lang.Expr
		var selectionType lang.Type
		if selection.SubSelect != nil {
			var err error
			selectionExpr, selectionType, err = s.getSubSelect(
				rowVarName, tableDesc, selection, typeScope, indexMap, depth,
			)
			if err != nil {
				return nil, err
			}
		} else {
			colDesc, err := tableDesc.getColDesc(selection.Name)
			if err != nil {
				return nil, fmt.Errorf("no such column: %s.%s", query.Table, selection.Name)
			}
			selectionExpr = lang.NewMemberAccess(lang.NewEVar(rowVarName), selection.Name)
			selectionType = colDesc.typ
		}

		exprs[selection.Name] = selectionExpr
		types[selection.Name] = selectionType
	}

	if join != nil {
		keyParam := rowVarName + "Key"
		selectionLambdaExpr := lang.NewELambda(
			[]lang.Param{
				{
					Typ:  tableDesc.getPKType(),
					Name: keyParam,
				},
			},
			lang.NewEDoBlock(
				[]lang.DoBinding{
					{
						rowVarName,
						lang.NewFuncCall(
							"get",
							[]lang.Expr{
								primaryIndexExpr,
								lang.NewEVar(keyParam),
							},
						),
					},
				},
				lang.NewERecord(exprs),
			),
			lang.NewTRecord(types),
		)

		joinIdxExpr := lang.NewEIndexRef(indexMap[join.manyTableName][join.manyJoinColName])

		// e.g. `get(comments.blog_post_id, blog_post.id)`
		subIndexExpr := lang.NewFuncCall("get", []lang.Expr{
			joinIdxExpr,
			lang.NewMemberAccess(lang.NewEVar(join.oneVarName), join.onePKName),
		})

		collectionExpr := lang.NewFuncCall("scan", []lang.Expr{
			subIndexExpr,
		})

		mapExpr := lang.NewFuncCall("map", []lang.Expr{
			collectionExpr,
			selectionLambdaExpr,
		})

		return mapExpr, nil
	}

	selectionLambdaExpr := lang.NewELambda(
		[]lang.Param{
			{
				Typ:  tableDesc.getType(),
				Name: rowVarName,
			},
		},
		lang.NewERecord(exprs),
		lang.NewTRecord(types),
	)

	// Scan the primary index...
	// TODO: maybe break this out into a function...
	collectionExpr := lang.NewFuncCall(
		"scan",
		[]lang.Expr{
			primaryIndexExpr,
		},
	)

	mapExpr := lang.NewFuncCall("map", []lang.Expr{
		collectionExpr,
		selectionLambdaExpr,
	})

	return mapExpr, nil
}

func (s *schema) getSubSelect(
	rowVarName string,
	tableDesc *tableDescriptor,
	selection *Selection,
	typeScope *lang.TypeScope,
	indexMap IndexMap,
	depth int,
) (lang.Expr, lang.Type, error) {
	// Make the scan of the table we're joining to.
	innerRowVarName := fmt.Sprintf("row%d", depth+1)

	// Just doing the has many case first.
	innerTableDesc := s.tables[selection.SubSelect.Table]
	colReferencingThis := innerTableDesc.colReferencingTable(tableDesc.name)
	if colReferencingThis == nil {
		return nil, nil, fmt.Errorf(
			"couldn't find a column on %s referencing %s", innerTableDesc.name, tableDesc.name,
		)
	}
	// Build filter
	equality := oneToManyJoin{
		oneVarName: rowVarName,
		onePKName:  tableDesc.primaryKey,

		manyTableName:   innerTableDesc.name,
		manyVarName:     innerRowVarName,
		manyJoinColName: *colReferencingThis,
	}

	innerTypeScope := typeScope.NewChildScope()
	innerTypeScope.Add(rowVarName, tableDesc.getType())

	subSelectExpr, err := s.planSelectInternal(
		selection.SubSelect, innerTypeScope, indexMap, depth+1, &equality,
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

type oneToManyJoin struct {
	oneVarName string
	onePKName  string

	manyTableName   string
	manyVarName     string
	manyJoinColName string
}
