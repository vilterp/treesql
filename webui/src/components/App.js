import React from "react";
import { Provider } from "react-redux";
import { createStore, applyMiddleware, combineReducers } from "redux";
import reducer from "../reducer";
import thunk from "redux-thunk";
import logger from "redux-logger";
import { startStatement, sendStatement, statementUpdate } from "../actions";
import { SCHEMA_QUERY } from "../lib/TreeSQLClient";
import REPL from "./REPL";
import "./App.css";

class App extends React.Component {
  componentWillMount() {
    const client = this.props.client;
    const store = createStore(
      combineReducers({
        state: reducer,
        client: (state, action) => client
      }),
      applyMiddleware(thunk, logger)
    );
    client.on("open", () => {
      store.dispatch(sendStatement(SCHEMA_QUERY + " live"));
    });
    client.on("statement_sent", channel => {
      store.dispatch(startStatement(channel.statementID, channel.statement));
      channel.on("update", update => {
        store.dispatch(statementUpdate(channel.statementID, update));
      });
    });

    function dispatchSocketState() {
      store.dispatch({
        type: "WEBSOCKET_STATE_TRANSITION",
        newState: client.readyState()
      });
    }

    client.on("close", dispatchSocketState);
    client.on("open", dispatchSocketState);
    client.on("error", dispatchSocketState);

    this.setState({
      store,
      list: [1, 2, 3]
    });

    setTimeout(() => {
      this.setState({
        list: [1, 2, 4, 3]
      });
    }, 1000);
  }

  render() {
    return (
      <Provider store={this.state.store}>
        <REPL />
      </Provider>
    );
  }
}

export default App;
