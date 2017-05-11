import _ from 'lodash';

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
    this.listeners[event].forEach((listener) => {
      listener(value);
    });
  }

}

export class TreeSQLClient extends EventEmitter {

  constructor(url) {
    this.super();
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
      const parsedMessage = JSON.parse(message);
      this.channels[parsedMessage.StatementID]._dispatchUpdate(parsedMessage.Message);
    });
  }

  readyState() {
    return this.websocket.readyState;
  }

  query(query) { // "command?"
    this.websocket.send(query);
    const channel = new Channel(this, this.nextStatementId);
    this.channels[this.nextStatementId] = channel;
    this.nextStatementId++;
    return channel;
  }

}

class Channel extends EventEmitter {

  constructor(client) {
    this.super();
    this.client = client;
  }

  _dispatchUpdate(message) {
    this._dispatch('update', message);
  }

}
