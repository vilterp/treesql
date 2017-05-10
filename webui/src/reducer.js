import immutable from 'dot-prop-immutable';

const initialState = {
  ui: {
    command: '',
    websocketState: WebSocket.CONNECTING
  },
  db: {
    messages: []
  }
};

let nextMessageId = 0;

export default function update(state = initialState, action) {
  switch (action.type) {
    case 'ADD_MESSAGE':
      return immutable.set(state, `db.messages.${nextMessageId++}`, {
        id: nextMessageId,
        source: action.source,
        message: action.message,
        timestamp: new Date()
      });
    
    case 'UPDATE_COMMAND':
      return immutable.set(state, 'ui.command', action.newValue);
    
    case 'WEBSOCKET_STATE_TRANSITION':
      return immutable.set(state, 'ui.websocketState', action.newState);

    default:
      return state;
  }
}
