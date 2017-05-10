import _ from 'lodash';
import React, { Component } from 'react';
import { connect } from 'react-redux';
import classNames from 'classnames';
import Autocomplete from 'react-autocomplete';
import { sendCommand } from '../actions.js';
import Message from './Message';
import './App.css';

const WEBSOCKET_STATES = {
  [WebSocket.CLOSED]: 'CLOSED',
  [WebSocket.CONNECTING]: 'CONNECTING',
  [WebSocket.OPEN]: 'OPEN',
  [WebSocket.CLOSING]: 'CLOSING'
}

class App extends Component {
  componentDidMount() {
    document.getElementById('command-input').focus(); // I forget how to do refs
  }

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
                <td><Message message={message.message} /></td>
                <td>{message.timestamp.toISOString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <form onSubmit={this.props.onSubmit}>
          <Autocomplete
            value={this.props.ui.command}
            onChange={(evt, value) => this.props.updateCommand(value)}
            onSelect={(value) => this.props.updateCommand(value)}
            items={_.reverse(this.props.ui.commandHistory)}
            shouldItemRender={(item, value) => (item.indexOf(value) !== -1)}
            renderItem={(item, isHighlighted) => (
              <div className={classNames('command-choice', { selected: isHighlighted })}>
                {item}
              </div>)}
            getItemValue={_.identity}
            inputProps={{ size: 100, id: 'command-input' }} />
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
