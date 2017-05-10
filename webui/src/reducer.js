import _ from 'lodash';
import immutable from 'dot-prop-immutable';
import { getCommandHistory } from './commandStorage';

const initialState = {
  ui: {
    command: '',
    websocketState: WebSocket.CONNECTING,
    commandHistory: getCommandHistory()
  },
  db: {
    messages: []
  }
};

let nextMessageId = 0;

export default function update(state = initialState, action) {
  switch (action.type) {
    case 'ADD_MESSAGE': {
      const addedToMessages = immutable.set(state, `db.messages.${nextMessageId++}`, {
        id: nextMessageId,
        source: action.source,
        message: action.message,
        timestamp: new Date()
      });
      // this should be split into a different reducer...
      if (action.source === 'client') {
        return {
          ...addedToMessages,
          ui: {
            ...state.ui,
            commandHistory: _.uniq([
              ...state.ui.commandHistory,
              action.message
            ])
          }
        }
      } else {
        return addedToMessages;
      }
    }
    case 'UPDATE_COMMAND':
      return immutable.set(state, 'ui.command', action.newValue);
    
    case 'WEBSOCKET_STATE_TRANSITION':
      return immutable.set(state, 'ui.websocketState', action.newState);

    default:
      return state;
  }
}
