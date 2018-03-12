import React from "react";
import classNames from 'classnames';
import "./RuleNameView.css";

export class RuleNameView extends React.Component {
  render() {
    return (
      <span
        className={classNames("rule-ref", {
          highlighted: this.props.id === this.props.highlightedRuleID
        })}
        onMouseOver={() => this.props.onHighlightRule(this.props.id, true)}
        onMouseOut={() => this.props.onHighlightRule(this.props.id, false)}
      >
        {this.props.name}
      </span>
    )
  }
}
