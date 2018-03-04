package treesql

import "testing"

func TestPlanFormat(t *testing.T) {
	blogPostsDesc := &TableDescriptor{
		Name:       "blog_posts",
		PrimaryKey: "id",
	}
	commentsDesc := &TableDescriptor{
		Name:       "comments",
		PrimaryKey: "id",
	}
	authorsDesc := &TableDescriptor{
		Name:       "authors",
		PrimaryKey: "id",
	}

	cases := []struct {
		node PlanNode
		exp  string
	}{
		{
			&FullScanNode{
				table: blogPostsDesc,
				selections: selections{
					selectColumns: []string{"id", "title"},
				},
			},
			`results0 = []
for row0 in blog_posts.indexes.id:
  result = {
    id: row0.id,
    title: row0.title,
  }
  results0.append(result)
return results0
`,
		},
		{
			&FullScanNode{
				table: blogPostsDesc,
				selections: selections{
					selectColumns: []string{"id", "title"},
					childNodes: map[string]PlanNode{
						"comments": &IndexScanNode{
							table:   commentsDesc,
							colName: "post_id",
							selections: selections{
								selectColumns: []string{"id", "body"},
							},
							matchExpr: Expr{
								Var: "id",
							},
						},
					},
				},
			},
			`results0 = []
for row0 in blog_posts.indexes.id:
  result = {
    id: row0.id,
    title: row0.title,
  }
  results1 = []
  for row1 in comments.indexes.post_id[row0.id]:
    result = {
      id: row1.id,
      body: row1.body,
    }
    results1.append(result)
  result.comments = results1
  results0.append(result)
return results0
`,
		},
		{
			&FullScanNode{
				table: blogPostsDesc,
				selections: selections{
					selectColumns: []string{"id", "title"},
					childNodes: map[string]PlanNode{
						"author": &IndexScanNode{
							table:   authorsDesc,
							colName: "id",
							selections: selections{
								selectColumns: []string{"name"},
							},
							matchExpr: Expr{
								Var: "author_id",
							},
						},
						"comments": &IndexScanNode{
							table:   commentsDesc,
							colName: "post_id",
							selections: selections{
								selectColumns: []string{"id", "body"},
							},
							matchExpr: Expr{
								Var: "id",
							},
						},
					},
				},
			},
			`results0 = []
for row0 in blog_posts.indexes.id:
  result = {
    id: row0.id,
    title: row0.title,
  }
  results1 = []
  for row1 in authors.indexes.id[row0.author_id]:
    result = {
      name: row1.name,
    }
    results1.append(result)
  result.author = results1
  results2 = []
  for row1 in comments.indexes.post_id[row0.id]:
    result = {
      id: row1.id,
      body: row1.body,
    }
    results2.append(result)
  result.comments = results1
  results0.append(result)
return results0
`,
		},
	}

	for idx, testCase := range cases {
		actual := FormatPlan(testCase.node)
		if actual != testCase.exp {
			t.Errorf("case %d:\nEXPECTED:\n\n%s\nGOT:\n\n%s\n", idx, testCase.exp, actual)
		}
	}
}
