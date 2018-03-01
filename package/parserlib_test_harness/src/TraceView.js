import React from "react";
import "./TraceView.css";
import "./GrammarView.css"; // factor out the common parts? idk
import { RuleNameView } from './RuleNameView';

function renderPos(pos) {
  return `${pos.Line}:${pos.Col}`
}

function renderSpan(trace) {
  return `${renderPos(trace.StartPos)} => ${renderPos(trace.EndPos)}`
}

export class TraceView extends React.Component {
  render() {
    return (
      <div className="trace-view">
        <TraceNode
          trace={this.props.trace}
          grammar={this.props.grammar}
          onHighlightRule={this.props.onHighlightRule}
          highlightedRuleID={this.props.highlightedRuleID}
        />
      </div>
    )
  }
}

class TraceNode extends React.Component {
  render() {
    const { trace, grammar } = this.props;

    if (!trace) {
      return <span>(empty)</span>;
    }

    const rule = grammar.RulesByID[trace.RuleID];
    switch (rule.RuleType) {
      case "SEQUENCE":
        return (
          <div>
            Sequence ({renderSpan(trace)})
            <ol style={{ marginTop: 0 }}>
              {trace.ItemTraces.map((itemTrace, idx) => (
                <li key={idx}>
                  <TraceNode
                    grammar={grammar}
                    trace={itemTrace}
                    onHighlightRule={this.props.onHighlightRule}
                    highlightedRuleID={this.props.highlightedRuleID}
                  />
                </li>
              ))}
            </ol>
          </div>
        );
      case "CHOICE":
        return (
          <div>
            Choice {trace.ChoiceIdx}<br/>
            <TraceNode
              grammar={grammar}
              trace={trace.ChoiceTrace}
              onHighlightRule={this.props.onHighlightRule}
              highlightedRuleID={this.props.highlightedRuleID}
            />
          </div>
        );
      case "REF": {
        return (
          <div>
            <RuleNameView
              id={grammar.TopLevelRules[rule.Ref]}
              name={rule.Ref}
              onHighlightRule={this.props.onHighlightRule}
              highlightedRuleID={this.props.highlightedRuleID}
            />
            <br />
            <TraceNode
              grammar={grammar}
              trace={trace.RefTrace}
              onHighlightRule={this.props.onHighlightRule}
              highlightedRuleID={this.props.highlightedRuleID}
            />
          </div>
        );
      }
      case "KEYWORD":
        // TODO: hover-ify
        return (
          <span className="rule-keyword">"{rule.Keyword}"</span>
        );
      case "REGEX":
        // TODO: hover-ify
        return (
          <span className="rule-regex">
            "{trace.RegexMatch.replace("\n", "\\n").replace("\t", "\\t")}"
          </span>
        );
      case "SUCCEED":
        return <span className="rule-succeed">&lt;succeed&gt;</span>;
      default:
        console.error(trace);
        return <pre>{JSON.stringify(trace)}</pre>
    }
  }

}
