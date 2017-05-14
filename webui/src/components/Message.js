import React from 'react';
import ReactJson from 'react-json-view'

class Message extends React.Component {

  shouldComponentUpdate(nextProps) {
    return nextProps.message !== this.props.message;
  }

  render() {
    const message = this.props.message;
    switch (message.type) {
      case 'error':
        return (
          <span className="message error">{message.error}</span>
        );
      case 'ack':
        return (
          <span className="message ack">{message.ack}</span>
        );
      case 'update':
        return (
          <div className="message update">
            <ReactJson
              src={message.update}
              displayDataTypes={false}
              displayObjectSize={false} />
          </div>
        );
      default:
        console.error('unknown message type', message);
        return null;
    }
  }

}

export default Message;
