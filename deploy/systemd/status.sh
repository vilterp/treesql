ssh -i $TREESQL_DEPLOY_KEY $TREESQL_DEPLOY_HOST 'systemctl status treesql-server'

echo
echo
echo

ssh -i $TREESQL_DEPLOY_KEY $TREESQL_DEPLOY_HOST 'systemctl status treesql-console'