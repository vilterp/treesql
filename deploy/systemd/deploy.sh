make webui
make treesql-server-linux
scp -i $TREESQL_DEPLOY_KEY ./treesql-server-linux $TREESQL_DEPLOY_HOST:treesql-server-next
scp -i $TREESQL_DEPLOY_KEY -r webui/build $TREESQL_DEPLOY_HOST:webui
ssh -i $TREESQL_DEPLOY_KEY $TREESQL_DEPLOY_HOST 'sudo systemctl stop treesql && mv treesql-server-next treesql-server && sudo systemctl start treesql'
