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
    const statement = getState().state.ui.statement;
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
  return (dispatch, getState) => {
    storeStatement(statement);
    getState().client.sendStatement(statement);
  };
}
