import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux'
import { createStore, applyMiddleware } from 'redux'
import reducer from './reducer';
import thunk from 'redux-thunk';
import logger from 'redux-logger';
import App from './components/App';
import TreeSQLClient, { SCHEMA_QUERY } from './lib/TreeSQLClient';
import { sendStatement } from './actions';
import './index.css';

const store = createStore(
  reducer,
  applyMiddleware(thunk, logger)
);

window.CLIENT = new TreeSQLClient(`ws://localhost:9000/ws`)
window.CLIENT.on('open', () => {
  store.dispatch(sendStatement(SCHEMA_QUERY + ' live'));
});

function dispatchSocketState() {
  store.dispatch({
    type: 'WEBSOCKET_STATE_TRANSITION',
    newState: window.CLIENT.readyState()
  });
}

window.CLIENT.on('close', dispatchSocketState);
window.CLIENT.on('open', dispatchSocketState);
window.CLIENT.on('error', dispatchSocketState);

// window.CLIENT.addEventListener('message', (msg) => {
//   const json = JSON.parse(msg.data);
//   store.dispatch(addMessage(json.Message, json.StatementID, 'server'));
// });

ReactDOM.render(
  <Provider store={store}>
    <App />
  </Provider>,
  document.getElementById('root')
);
