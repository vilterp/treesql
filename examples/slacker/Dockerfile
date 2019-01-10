FROM node:6.10

COPY . slacker
WORKDIR slacker
RUN npm install
RUN npm run build

RUN npm install -g http-server

CMD http-server build -p $PORT -d false
