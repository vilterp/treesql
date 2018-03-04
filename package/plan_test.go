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
			`blog_posts_results = []
for row in blog_posts.indexes.id:
  blog_posts_result = {
    id: row.id,
    title: row.title,
  }
  blog_posts_results.append(blog_posts_result)
return blog_posts_results
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
			`blog_posts_results = []
for row in blog_posts.indexes.id:
  blog_posts_result = {
    id: row.id,
    title: row.title,
  }
  # comments
  comments_results = []
  for row in comments.indexes.post_id[row.id]:
    comments_result = {
      id: row.id,
      body: row.body,
    }
    comments_results.append(comments_result)
  blog_posts_result.comments = comments_results
  blog_posts_results.append(blog_posts_result)
return blog_posts_results
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
			`blog_posts_results = []
for row in blog_posts.indexes.id:
  blog_posts_result = {
    id: row.id,
    title: row.title,
  }
  # author
  authors_results = []
  for row in authors.indexes.id[row.author_id]:
    authors_result = {
      name: row.name,
    }
    authors_results.append(authors_result)
  blog_posts_result.author = authors_results
  # comments
  comments_results = []
  for row in comments.indexes.post_id[row.id]:
    comments_result = {
      id: row.id,
      body: row.body,
    }
    comments_results.append(comments_result)
  blog_posts_result.comments = comments_results
  blog_posts_results.append(blog_posts_result)
return blog_posts_results
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
