export function sendCommand() {
  return (dispatch, getState) => {
    const command = getState().ui.command;
    dispatch(addMessage(command, 'client'))
    dispatch({
      type: 'UPDATE_COMMAND',
      newValue: ''
    });
    window.SOCKET.send(command);
  };
}

export function addMessage(message, source) {
  var maybeJSON;
  try {
    maybeJSON = JSON.parse(message)
  } catch (e) {
    maybeJSON = message
  }
  return {
    type: 'ADD_MESSAGE',
    message: maybeJSON,
    source
  };
}
