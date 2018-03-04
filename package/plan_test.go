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

	// TODO: these are hilariously verbose and repetitive
	// but I like the fact that you can see exactly what they'll do
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
			`[
  for row in blog_posts.by_id {
    yield {
      id: row.id,
      title: row.title,
    }
  }
]
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
			`[
  for row in blog_posts.by_id {
    yield {
      id: row.id,
      title: row.title,
      comments: [
        for row in comments.by_post_id[row.id] {
          yield {
            id: row.id,
            body: row.body,
          }
        }
      ],
    }
  }
]
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
			`[
  for row in blog_posts.by_id {
    yield {
      id: row.id,
      title: row.title,
      author: [
        for row in authors.by_id[row.author_id] {
          yield {
            name: row.name,
          }
        }
      ],
      comments: [
        for row in comments.by_post_id[row.id] {
          yield {
            id: row.id,
            body: row.body,
          }
        }
      ],
    }
  }
]
`,
		},
	}

	// TODO: case with some WHEREs

	for idx, testCase := range cases {
		actual := FormatPlan(testCase.node)
		if actual != testCase.exp {
			t.Errorf("case %d:\nEXPECTED:\n\n%s\nGOT:\n\n%s\n", idx, testCase.exp, actual)
		}
	}
}
