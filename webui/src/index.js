import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux'
import { createStore, applyMiddleware } from 'redux'
import reducer from './reducer';
import thunk from 'redux-thunk';
import logger from 'redux-logger';
import App from './components/App';
import TreeSQLClient, { SCHEMA_QUERY } from './lib/TreeSQLClient';
import {
  startStatement,
  sendStatement,
  statementUpdate
} from './actions';
import './index.css';

const store = createStore(
  reducer,
  applyMiddleware(thunk, logger)
);

window.CLIENT = new TreeSQLClient(`ws://localhost:9000/ws`)
window.CLIENT.on('open', () => {
  store.dispatch(sendStatement(SCHEMA_QUERY + ' live'));
});
window.CLIENT.on('statement_sent', (channel) => {
  store.dispatch(startStatement(channel.statementID, channel.statement));
  channel.on('update', (update) => {
    store.dispatch(statementUpdate(channel.statementID, update));
  });
})

function dispatchSocketState() {
  store.dispatch({
    type: 'WEBSOCKET_STATE_TRANSITION',
    newState: window.CLIENT.readyState()
  });
}

window.CLIENT.on('close', dispatchSocketState);
window.CLIENT.on('open', dispatchSocketState);
window.CLIENT.on('error', dispatchSocketState);

ReactDOM.render(
  <Provider store={store}>
    <div>
      <App />
    </div>
  </Provider>,
  document.getElementById('root')
);
