import React from "react";
import _ from "lodash";
import { RuleNameView } from './RuleNameView';
import classNames from "classnames";
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
        <tbody>
          {_.map(grammar.TopLevelRules, (ruleID, name) => (
            <tr key={name}>
              <td>
                <RuleNameView
                  onHighlightRule={this.props.onHighlightRule}
                  id={ruleID}
                  name={name}
                  highlightedRuleID={this.props.highlightedRuleID}
                />
              </td>
              <td>
                <RuleView
                  ruleID={ruleID}
                  grammar={grammar}
                  onHighlightRule={this.props.onHighlightRule}
                  highlightedRuleID={this.props.highlightedRuleID}
                />
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
    const {
      ruleID,
      grammar,
      onHighlightRule,
      highlightedRuleID,
    } = this.props;

    const rule = grammar.RulesByID[ruleID];

    const highlightProps = {
      onHighlightRule: onHighlightRule,
      highlightedRuleID: highlightedRuleID,
    };

    const isHighlighted = ruleID === highlightedRuleID;

    function highlightWrapper(element) {
      return (
        <span
          className={classNames("rule-def", { highlighted: isHighlighted })}
          onMouseEnter={() => onHighlightRule(ruleID, true)}
          onMouseLeave={() => onHighlightRule(ruleID, false)}
        >
          {element}
        </span>
      );
    }

    switch (rule.RuleType) {
      case "SEQUENCE": {
        return (
          highlightWrapper(
            <span>
              <span className="rule-symbol">[</span>
              {intersperse(
                rule.SeqItems.map((ruleID, idx) => (
                  <span key={`item-${idx}`}>
                    <RuleView
                      ruleID={ruleID}
                      grammar={grammar}
                      {...highlightProps}
                    />
                  </span>
                )),
                (i) => <span key={i}>, </span>,
              )}
              <span className="rule-symbol">]</span>
            </span>
          )
        );
      }
      case "CHOICE":
        return (
          highlightWrapper(
            <span>
            {intersperse(
              rule.Choices.map((ruleID, idx) => (
                <span key={`item-${idx}`}>
                  <RuleView
                    ruleID={ruleID}
                    grammar={grammar}
                    {...highlightProps}
                  />
                </span>
              )),
              (i) => <span key={i} className="rule-symbol"> | </span>,
            )}
          </span>
          )
        );
      case "KEYWORD":
        return highlightWrapper(<span className="rule-keyword">"{rule.Keyword}"</span>);
      case "REGEX":
        return highlightWrapper(<span className="rule-regex">/${rule.Regex}/</span>);
      case "SUCCEED":
        return <span className="rule-succeed">&lt;succeed&gt;</span>;
      case "REF":
        return (
          <RuleNameView
            id={grammar.TopLevelRules[rule.Ref]}
            name={rule.Ref}
            {...highlightProps}
          />
        );
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
