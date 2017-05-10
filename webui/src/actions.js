export function sendCommand() {
  return (dispatch, getState) => {
    const command = getState().ui.command;
    dispatch({
      type: 'ADD_MESSAGE',
      message: command,
      source: 'client'
    });
    dispatch({
      type: 'UPDATE_COMMAND',
      newValue: ''
    });
    window.SOCKET.send(command);
  };
}
