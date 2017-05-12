import _ from 'lodash';
import immutable from 'object-path-immutable';
import { getStatementHistory } from './statementStorage';

const initialState = {
  ui: {
    statement: '',
    websocketState: WebSocket.CONNECTING,
    statementHistory: getStatementHistory()
  },
  statements: []
};

export default function update(state = initialState, action) {
  // TODO: really have to figure out how to use dot-prop-immutable or similar
  // to push onto an array. jeez
  switch (action.type) {
    case 'START_STATEMENT': {
      const newState1 = immutable.push(state, 'statements', {
        id: action.statementID,
        statement: action.statement,
        updates: []
      });
      return immutable.push(newState1, 'ui.statementHistory', action.statement);
    }
    case 'STATEMENT_UPDATE':
      // assumption that index = statement id could be off maybe?
      return immutable.push(state, `statements.${action.statementID}.updates`, action.update);

    case 'UPDATE_STATEMENT':
      return immutable.set(state, 'ui.statement', action.newValue);
    
    case 'WEBSOCKET_STATE_TRANSITION':
      return immutable.set(state, 'ui.websocketState', action.newState);

    default:
      return state;
  }
}
