FROM node:6.10

COPY . slacker
WORKDIR slacker
RUN yarn --no-progress
RUN yarn build

RUN yarn global add http-server

CMD http-server build -p $PORT -d false
