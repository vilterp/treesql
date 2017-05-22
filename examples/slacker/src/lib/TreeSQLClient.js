import _ from 'lodash';

export const SCHEMA_QUERY = `
  many __tables__ {
    name,
    primary_key,
    columns: many __columns__ {
      id,
      name,
      type,
      references
    }
  }
`;

class EventEmitter {

  constructor() {
    this.listeners = {};
  }

  // TODO: auto-reconnect

  on(event, listener) {
    var listeners = this.listeners[event];
    if (!listeners) {
      listeners = [];
      this.listeners[event] = listeners;
    }
    listeners.push(listener);
  }

  off(event, listener) {
    _.remove(this.listeners[event], listener);
  }

  _dispatch(event, value) {
    const listeners = this.listeners[event];
    if (listeners) {
      this.listeners[event].forEach((listener) => {
        listener(value);
      });
    }
  }

}

class Channel extends EventEmitter {

  constructor(client, statement, statementID) {
    super();
    this.client = client;
    this.statement = statement;
    this.statementID = statementID;
  }

  _dispatchUpdate(message) {
    this._dispatch('update', message);
  }

}

export default class TreeSQLClient extends EventEmitter {

  constructor(url) {
    super();
    this.nextStatementId = 0;
    this.channels = {};
    this.websocket = new WebSocket(url);
    this.websocket.addEventListener('open', (evt) => {
      this._dispatch('open', evt);
    });
    this.websocket.addEventListener('close', (evt) => {
      this._dispatch('close', evt);
    });
    this.websocket.addEventListener('error', (evt) => {
      this._dispatch('error', evt);
    });
    this.websocket.addEventListener('message', (message) => {
      const parsedMessage = JSON.parse(message.data);
      this.channels[parsedMessage.StatementID]._dispatchUpdate(parsedMessage.Message);
    });
  }

  readyState() {
    return this.websocket.readyState;
  }

  sendStatement(statement) {
    this.websocket.send(statement);
    const channel = new Channel(this, statement, this.nextStatementId);
    this.channels[this.nextStatementId] = channel;
    this._dispatch('statement_sent', channel);
    this.nextStatementId++;
    return channel;
  }

}
