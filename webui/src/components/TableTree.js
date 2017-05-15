import _ from 'lodash';
import React from 'react';
import './TableTree.css';

function Cell({ cell }) {
  return _.isString(cell)
    ? <td>{cell}</td>
    : <td className="cell cell-unexpanded">({cell.length} items)</td>;
}

class TableTree extends React.Component {

  render() {
    if (this.props.records.length == 0) {
      return (
        <div className="table-tree empty">(empty)</div>
      );
    }
    const columns = _.keys(this.props.records[0]);
    return (
      <table className="table-tree">
        <thead>
          <tr>
            {columns.map((column, idx) => (
              <th key={idx}>{column}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {this.props.records.map((record, idx) => (
            <tr key={idx}>
              {columns.map((column, colIdx) => (
                <Cell key={colIdx} cell={record[column]} />
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    );
  }

}

export default TableTree;
