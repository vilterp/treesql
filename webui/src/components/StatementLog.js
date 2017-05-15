import React from 'react';
import Message from './Message';
import './StatementLog.css';

function StatementLog({ updates }) {
  return (
    <table className="statement-log">
      <thead>
        <tr>
          <th style={{ width: 100 }}>Type</th>
          <th>Message</th>
          <th style={{ width: 200 }}>Timestamp</th>
        </tr>
      </thead>
      <tbody>
        {updates.map((message, idx) => (
          <tr key={idx} className={`message message-${message.type}`}>
            <td className="statement-log-message-type">{message.type}</td>
            <td>
              <Message message={message} />
            </td>
            <td className="statement-log-timestamp">{message.timestamp.toISOString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default StatementLog;
