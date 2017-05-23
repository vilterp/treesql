ssh -i $TREESQL_DEPLOY_KEY $TREESQL_DEPLOY_HOST 'sudo systemctl stop treesql-server; sudo systemctl start treesql-server'
ssh -i $TREESQL_DEPLOY_KEY $TREESQL_DEPLOY_HOST 'sudo systemctl stop treesql-console; sudo systemctl start treesql-console'
