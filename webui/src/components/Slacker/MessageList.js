import React from 'react';

function MessageList({ messages }) {
  return (
    <div>
      <table>
        {messages.map((message) => (
          <tr key={message.id}>
            <td>{message.timestamp.toISOString()}</td>
            <td>{message.user.name}</td>
            <td>{message.body}</td>
          </tr>
        ))}
      </table>
      <form>
        <input type="text" size="100" />
        <button>Send</button>
      </form>
    </div>
  );
}

export default MessageList;
