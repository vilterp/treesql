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
    statement
      .split(';')
      .filter((stmt) => (stmt.length > 0))
      .forEach((splitStatement) => {
        dispatch(sendStatement(splitStatement));
      });
  };
}

export function sendStatement(statement) {
  return (dispatch) => {
    storeStatement(statement);
    window.CLIENT.sendStatement(statement);
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
