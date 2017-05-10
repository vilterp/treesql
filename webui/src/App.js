import React, { Component } from 'react';
import { connect } from 'react-redux';
// import JSONTree from 'react-json-tree'
import ReactJson from 'react-json-view'
import { sendCommand } from './actions.js';
import './App.css';

const WEBSOCKET_STATES = {
  [WebSocket.CLOSED]: 'CLOSED',
  [WebSocket.CONNECTING]: 'CONNECTING',
  [WebSocket.OPEN]: 'OPEN',
  [WebSocket.CLOSING]: 'CLOSING'
}

class App extends Component {
  render() {
    return (
      <div className="App">
        <div>
          Websocket state: {WEBSOCKET_STATES[this.props.ui.websocketState]}
        </div>
        <table id="messages">
          <thead>
            <tr>
              <td>Source</td>
              <td>Message</td>
              <td>Timestamp</td>
            </tr>
          </thead>
          <tbody>
            {this.props.db.messages.map((message, idx) => (
              <tr key={idx} className={`source-${message.source}`}>
                <td>{message.source}</td>
                <td>{typeof(message.message) === 'object'
                  ? <ReactJson
                      src={message.message}
                      displayDataTypes={false}
                      displayObjectSize={false} />
                  : message.message}</td>
                <td>{message.timestamp.toISOString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <form onSubmit={this.props.onSubmit}>
          <input
            id="command-input"
            type="text"
            size="50"
            autoComplete="off"
            onInput={(evt) => this.props.updateCommand(evt.target.value)}
            value={this.props.ui.command} />
          <button>Send</button>
        </form>
      </div>
    );
  }
}

function mapStateToProps(state) {
  return state;
}

function mapDispatchToProps(dispatch) {
  return {
    onSubmit: (evt) => {
      evt.preventDefault();
      dispatch(sendCommand());
    },
    updateCommand: (newValue) => {
      dispatch({
        type: 'UPDATE_COMMAND',
        newValue
      })
    }
  };
}

export default connect(mapStateToProps, mapDispatchToProps)(App);
