import React from 'react';
import ReactDOM from 'react-dom';
import App from './components/App';
import TreeSQLClient from './lib/TreeSQLClient';
import './index.css';

const client = new TreeSQLClient(`ws://localhost:9000/ws`)

ReactDOM.render(
  <App client={client} />,
  document.getElementById('root')
);
