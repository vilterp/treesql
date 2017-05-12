import { storeStatement } from './statementStorage';

export const START_STATEMENT = 'START_STATEMENT';
export const startStatement = (statementID, statement) => ({
  type: START_STATEMENT,
  statementID,
  statement
});

export const STATEMENT_UPDATE = 'STATEMENT_UPDATE';
export const statementUpdate = (statementID, update) => ({
  type: STATEMENT_UPDATE,
  statementID,
  update
});

export function sendStatementFromInput() {
  return (dispatch, getState) => {
    const statement = getState().ui.statement;
    dispatch({
      type: 'UPDATE_STATEMENT',
      newValue: ''
    });
    dispatch(sendStatement(statement));
  };
}

export function sendStatement(statement) {
  return (dispatch) => {
    storeStatement(statement);
    const channel = window.CLIENT.sendStatement(statement);
    dispatch(startStatement(channel.statementID, statement));
    channel.on('update', (update) => {
      dispatch(statementUpdate(channel.statementID, update));
    });
  };
}

// export function addMessage(message, statementId, source) {
//   return function(dispatch) {
//     var maybeJSON;
//     try {
//       maybeJSON = JSON.parse(message)
//     } catch (e) {
//       maybeJSON = message
//     }
//     dispatch({
//       type: 'ADD_MESSAGE',
//       message: maybeJSON,
//       statementId,
//       source
//     });
//     if (source === 'server') {
//       window.scrollTo(0, document.body.scrollHeight - window.innerHeight - 250);
//     }
//   }
// }
