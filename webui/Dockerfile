FROM node:6.10

COPY . webui
WORKDIR webui
RUN npm install
RUN npm run build

RUN npm install -g http-server

EXPOSE 8080

CMD http-server build -p 8080 -d false
