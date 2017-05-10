import { storeCommand } from './commandStorage';

export function sendCommand() {
  return (dispatch, getState) => {
    const command = getState().ui.command;
    dispatch(addMessage(command, getState().ui.nextStatementId, 'client'));
    storeCommand(command);
    dispatch({
      type: 'UPDATE_COMMAND',
      newValue: ''
    });
    window.SOCKET.send(command);
  };
}

export function addMessage(message, statementId, source) {
  return function(dispatch) {
    var maybeJSON;
    try {
      maybeJSON = JSON.parse(message)
    } catch (e) {
      maybeJSON = message
    }
    dispatch({
      type: 'ADD_MESSAGE',
      message: maybeJSON,
      statementId,
      source
    });
    if (source === 'server') {
      window.scrollTo(0, document.body.scrollHeight - window.innerHeight - 250);
    }
  }
}
