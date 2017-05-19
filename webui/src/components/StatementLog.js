import _ from 'lodash';
import React from 'react';
import Message from './Message';
import './StatementLog.css';

class StatementLog extends React.Component {

  constructor() {
    super();
    this.toggleExpansion = this.toggleExpansion.bind(this);
  }

  componentWillMount() {
    this.setState({
      collapsed: {}
    });
  }

  toggleExpansion(idx) {
    this.setState({
      collapsed: {
        [idx]: this.isExpanded(idx)
      }
    });
  }

  isExpanded(idx) {
    const collapsed = this.state.collapsed[idx];
    return !collapsed;
  }

  render() {
    const { updates } = this.props;
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
              <td className="statement-log-message-type">
                <span
                  className={`expansion-toggle ${this.isExpanded(idx) ? 'expanded' : 'non-expanded'}`}
                  onClick={() => this.toggleExpansion(idx)}>
                  {message.type}
                </span>
              </td>
              <td>
                {this.isExpanded(idx)
                  ? <Message message={message} />
                  : <span
                      className="expansion-toggle"
                      onClick={() => this.toggleExpansion(idx)}>
                      (Collapsed)</span>
                }
              </td>
              <td className="statement-log-timestamp">{message.timestamp.toISOString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    );
  }

}

export default StatementLog;
