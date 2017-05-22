import _ from 'lodash';
import React from 'react';
import './TableTree.css';

function Cell({ cell, expanded, setExpanded }) {
  return _.isString(cell)
    ? <td>{cell}</td>
    : expanded
      // maybe collapse these down idk
      ? <td className="cell cell-expanded">
          <span
            className="expansion-link"
            onClick={() => setExpanded(false)}>
            ({cell.length} items)
          </span>
        </td>
      : <td className="cell cell-unexpanded">
          <span
            className="expansion-link"
            onClick={() => setExpanded(true)}>
            ({cell.length} items)
          </span>
        </td>;
}

class TableTree extends React.Component {

  componentWillMount() {
    const expansions = _.mapValues(this.props.records, _.constant(null));
    this.setState({
      expansions
    });
  }

  render() {
    if (this.props.records.length === 0) {
      return (
        <div className="table-tree empty">(empty)</div>
      );
    }
    const setExpandedForCell = (recordIdx, column) => {
      return ((expanded) => {
        this.setState({
          expansions: _.merge({}, this.state.expansions, {
            [recordIdx]: expanded ? column : null
          })
        })
      });
    }
    // could compute this on mount & cache in state
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
          {_.flatten(this.props.records.map((record, idx) => {
            const recordRow = (
              <tr key={idx} className="records-row">
                {columns.map((column, colIdx) => (
                  <Cell
                    key={colIdx}
                    cell={record[column]}
                    setExpanded={setExpandedForCell(idx, column)}
                    expanded={this.state.expansions[idx] === column} />
                ))}
              </tr>
            );
            const expandedColumn = this.state.expansions[idx];
            if (expandedColumn === null) {
              return [recordRow];
            } else {
              const expansionsRow = (
                <tr key={`${idx}-expansions`} className="expansions-row">
                  <td colSpan={columns.length}>
                    {record[expandedColumn]
                      ? <TableTree records={record[expandedColumn]} />
                      : <span>wut</span> /* TODO why is this hit sometimes */}
                  </td>
                </tr>
              );
              return [
                recordRow,
                expansionsRow
              ];
            }
          }))}
        </tbody>
      </table>
    );
  }

}

export default TableTree;
