import React from 'react';
import ReactJson from 'react-json-view'

class Message extends React.Component {

  shouldComponentUpdate(nextProps) {
    return nextProps.message !== this.props.message;
  }

  render() {
    const message = this.props.message;
    return typeof(message) === 'object'
        ? <ReactJson
            src={message}
            displayDataTypes={false}
            displayObjectSize={false} />
        : <span>{message}</span>;
  }

}

export default Message;
