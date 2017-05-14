import React from 'react';
import ReactJson from 'react-json-view'

class Message extends React.Component {

  shouldComponentUpdate(nextProps) {
    return nextProps.message !== this.props.message;
  }

  render() {
    // TODO: table with timestamp & type
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
      case 'initial_result':
      case 'table_update':
      case 'record_update':
        return (
          <div className="message update">
            <strong>{message.type}:</strong><br />
            <ReactJson
              src={message.payload}
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
