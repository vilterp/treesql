import React from "react";
import _ from "lodash";
import "./GrammarView.css";

export class GrammarView extends React.Component {

  render() {
    const grammar = this.props.grammar;

    return (
      <table className="grammar-view">
        <thead>
          <tr>
            <th>Name</th>
            <th>Definition</th>
          </tr>
        </thead>
        <tbody style={{ fontFamily: "monospace" }}>
          {_.map(grammar.TopLevelRules, (ruleID, name) => (
            <tr key={name}>
              <td className="rule-ref">{name}</td>
              <td>
                <RuleView ruleID={ruleID} grammar={grammar} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    )
  }

}

class RuleView extends React.Component {

  render() {
    const ruleID = this.props.ruleID;
    const grammar = this.props.grammar;
    const rule = grammar.RulesByID[ruleID];

    if (!rule) {
      return <span>nil rule???</span>
    }

    switch (rule.RuleType) {
      case "SEQUENCE": {
        return (
          <span>
            <span className="rule-symbol">[</span>
            {intersperse(
              rule.SeqItems.map((ruleID, idx) => (
                <span key={`item-${idx}`}>
                  <RuleView ruleID={ruleID} grammar={grammar} />
                </span>
              )),
              (i) => <span key={i}>, </span>,
            )}
            <span className="rule-symbol">]</span>
          </span>
        );
      }
      case "CHOICE":
        return (
          <span>
            {intersperse(
              rule.Choices.map((ruleID, idx) => (
                <span key={`item-${idx}`}>
                  <RuleView ruleID={ruleID} grammar={grammar} />
                </span>
              )),
              (i) => <span key={i} className="rule-symbol"> | </span>,
            )}
          </span>
        );
      case "KEYWORD":
        return <span className="rule-keyword">"{rule.Keyword}"</span>;
      case "REGEX":
        return <span className="rule-regex">/${rule.Regex}/</span>;
      case "SUCCEED":
        return <span className="rule-succeed">&lt;succeed&gt;</span>;
      case "REF":
        // TODO: hover-ify
        return <span className="rule-ref">{rule.Ref}</span>;
      default:
        return JSON.stringify(rule);
    }
  }

}

// e.g. intersperse(["foo", "bar", "baz"], "-") => ["foo", "-", "bar", "-", "baz"]
// TODO: move to util
function intersperse(array, sep) {
  const output = [];
  for (let i = 0; i < array.length; i++) {
    if (i > 0) {
      output.push(sep(i));
    }
    output.push(array[i]);
  }
  return output;
}
