import React from 'react';
import TableTree from './TableTree';
import ReactJson from 'react-json-view';

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
        return (
          <div className="message update">
            <TableTree records={message.initial_result.Data} />
          </div>
        );
      case 'table_update':
        return (
          <ReactJson
            src={message.table_update}
            displayDataTypes={false}
            displayObjectSize={false} />
        );
      case 'record_update':
        return (
          <ReactJson
            src={message.record_update}
            displayDataTypes={false}
            displayObjectSize={false} />
        );
      default:
        console.error('unknown message type', message);
        return null;
    }
  }

}

export default Message;
