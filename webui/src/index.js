import React from "react";
import ReactDOM from "react-dom";
import App from "./components/App";
import TreeSQLClient from "./lib/TreeSQLClient";
import "./index.css";

const url = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}:${window.location.port}/ws`;

const client = new TreeSQLClient(url);

ReactDOM.render(<App client={client} />, document.getElementById("root"));

console.log('hello from TreeSQL console');
