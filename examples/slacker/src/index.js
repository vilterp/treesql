import React from 'react';
import ReactDOM from 'react-dom';
import registerServiceWorker from './registerServiceWorker';
import { Provider } from 'react-redux'
import { createStore, applyMiddleware } from 'redux'
import thunk from 'redux-thunk';
import logger from 'redux-logger';

import TreeSQLClient from './lib/TreeSQLClient';
import liveQueryReducer from './lib/liveQueryReducer';
import { updateToAction } from './lib/liveQueryActions';

import App from './App';
import { QUERY } from './components/Slacker';
import './index.css';

const store = createStore(
  liveQueryReducer,
  applyMiddleware(thunk, logger)
);

window.CLIENT = new TreeSQLClient(`ws://${window.location.host}:9000/ws`)
window.CLIENT.on('open', () => {
  const channel = window.CLIENT.sendStatement(QUERY);
  channel.on('update', (update) => {
    const action = updateToAction(update);
    if (action) {
      store.dispatch(action);
    }
  });
});

ReactDOM.render(
  <Provider store={store}>
    <div>
      <App />
    </div>
  </Provider>,
  document.getElementById('root')
);

registerServiceWorker();
