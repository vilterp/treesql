import _ from 'lodash';
import immutable from 'dot-prop-immutable';
import { getCommandHistory } from './commandStorage';

const initialState = {
  ui: {
    command: '',
    websocketState: WebSocket.CONNECTING,
    commandHistory: getCommandHistory(),
    nextStatementId: 0
  },
  db: {
    messages: []
  }
};

export default function update(state = initialState, action) {
  switch (action.type) {
    case 'ADD_MESSAGE': {
      const addedToMessages = {
        ...state,
        db: {
          ...state.db,
          messages: [
            ...state.db.messages,
            {
              source: action.source,
              message: action.message,
              timestamp: new Date(),
              statementId: action.statementId
            }
          ]
        }
      }
      // this should be split into a different reducer...
      if (action.source === 'client') {
        return {
          ...addedToMessages,
          ui: {
            ...state.ui,
            commandHistory: _.uniq([
              ...state.ui.commandHistory,
              action.message
            ]),
            nextStatementId: state.ui.nextStatementId + 1
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
