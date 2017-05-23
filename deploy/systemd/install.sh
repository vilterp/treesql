scp -i $TREESQL_DEPLOY_KEY treesql-console.service $TREESQL_DEPLOY_HOST:/home/ubuntu
scp -i $TREESQL_DEPLOY_KEY treesql-server.service $TREESQL_DEPLOY_HOST:/home/ubuntu
ssh -i $TREESQL_DEPLOY_KEY $TREESQL_DEPLOY_HOST 'sudo mv treesql-server.service /etc/systemd/system && sudo mv treesql-console.service /etc/systemd/system && sudo systemctl daemon-reload'
scp -i $TREESQL_DEPLOY_KEY ran $TREESQL_DEPLOY_HOST:/home/ubuntu
