import React from 'react';

class Room extends React.Component {

  constructor() {
    super();
    this.sendMessage = this.sendMessage.bind(this);
  }

  componentWillMount() {
    this.setState({ message: '' });
  }

  sendMessage() {
    // createtable messages (
    //   id string primarykey,
    //   room_id string referencestable rooms,
    //   user_id string referencestable users,
    //   timestamp string,
    //   body string
    // )
    // uh, yeah. sending kv pairs would be nice
    const messageID = Math.random();
    const timestamp = new Date().getTime();
    window.CLIENT.sendStatement(`INSERT INTO messages VALUES (
      "${messageID}",
      "${this.props.room.id}",
      "${this.props.currentUserID}",
      "${timestamp}",
      "${this.state.message}"
    )`);
    this.setState({
      message: ''
    });
  }

  render() {
    return (
      <div>
        <table>
          <tbody>
            {this.props.room.messages.map((message) => (
              <tr key={message.id}>
                <td>{new Date(parseInt(message.timestamp, 10)).toISOString()}</td>
                <td>{message.user[0].name}</td>{/*TODO: ONE*/}
                <td>{message.body}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <form>
          <input
            type="text"
            size="100"
            value={this.state.message}
            onInput={(evt) => { this.setState({ message: evt.target.value }); }} />
          <button
            onClick={(evt) => { evt.preventDefault(); this.sendMessage(); }}>
            Send
          </button>
        </form>
      </div>
    );
  }

}

export default Room;
