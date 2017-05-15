import React from 'react';
import TableTree from './TableTree';

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
          <span className="message error">{message.payload}</span>
        );
      case 'ack':
        return (
          <span className="message ack">{message.payload}</span>
        );
      case 'initial_result':
      case 'table_update':
      case 'record_update':
        return (
          <div className="message update">
            <TableTree records={message.payload} />
          </div>
        );
      default:
        console.error('unknown message type', message);
        return null;
    }
  }

}

export default Message;
