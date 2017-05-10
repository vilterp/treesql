import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux'
import { createStore, applyMiddleware } from 'redux'
import App from './App';
import reducer from './reducer';
import thunk from 'redux-thunk';
import logger from 'redux-logger';
import './index.css';

window.SOCKET = new WebSocket('ws://localhost:9000/ws'); // TODO: make this configurable I guess

const store = createStore(
  reducer,
  applyMiddleware(thunk, logger)
);

function dispatchSocketState() {
  store.dispatch({
    type: 'WEBSOCKET_STATE_TRANSITION',
    newState: window.SOCKET.readyState
  });
}

window.SOCKET.addEventListener('close', dispatchSocketState);
window.SOCKET.addEventListener('open', dispatchSocketState);
window.SOCKET.addEventListener('error', dispatchSocketState);

window.SOCKET.addEventListener('message', (msg) => {
  store.dispatch({
    type: 'ADD_MESSAGE',
    source: 'server',
    message: msg.data
  });
});

ReactDOM.render(
  <Provider store={store}>
    <App />
  </Provider>,
  document.getElementById('root')
);
