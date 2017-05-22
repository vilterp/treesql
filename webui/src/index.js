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

// TODO: move into own app
import { QUERY } from './components/Slacker/Slacker';
import Container from './components/Slacker/Container';
import liveQueryReducer from './lib/liveQueryReducer';
import { updateToAction } from './lib/liveQueryActions';

const store = createStore(
  (state, action) => liveQueryReducer(
    reducer(
      state,
      action
    ),
    action
  ),
  applyMiddleware(thunk, logger)
);

// TODO: move this somewhere (probably a library of some kind)
function initializeSlacker() {
  const channel = window.CLIENT.sendStatement(QUERY);
  channel.on('update', (update) => {
    console.log('Slacker update:', update);
    const action = updateToAction(update);
    if (action) {
      store.dispatch(action);
    }
  });
}

window.CLIENT = new TreeSQLClient(`ws://localhost:9000/ws`)
window.CLIENT.on('open', () => {
  store.dispatch(sendStatement(SCHEMA_QUERY + ' live'));
  initializeSlacker();
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

ReactDOM.render(
  <Provider store={store}>
    <div>
      <Container />
      <App />
    </div>
  </Provider>,
  document.getElementById('root')
);
