import React from "react";
import "./TraceView.css";
import "./GrammarView.css"; // factor out the common parts? idk
import { RuleNameView } from './RuleNameView';
import { formatSpan } from './span';
import classNames from "classnames";
import "./SourceView.css";

export class TraceView extends React.Component {
  render() {
    return (
      <div className="trace-view">
        <TraceNode {...this.props} />
      </div>
    )
  }
}

class TraceNode extends React.Component {
  render() {
    const {
      trace,
      grammar,
      onHighlightSpan,
      highlightedSpan,
    } = this.props;

    if (!trace) {
      return <span>(empty)</span>;
    }

    const highlightProps = {
      onHighlightRule: this.props.onHighlightRule,
      highlightedRuleID: this.props.highlightedRuleID,
      onHighlightSpan: onHighlightSpan,
      highlightedSpan: highlightedSpan,
    };

    const formattedSpan = formatSpan(trace);
    const isHighlightedSpan = formattedSpan === highlightedSpan;

    function highlightWrapper(className, element) {
      return (
        <span
          className={classNames(
            className, "source-span",
            { highlighted: isHighlightedSpan },
          )}
          onMouseOver={() => onHighlightSpan(formattedSpan, true)}
          onMouseOut={() => onHighlightSpan(formattedSpan, false)}
        >
          {element}
        </span>
      )
    }

    const rule = grammar.RulesByID[trace.RuleID];
    switch (rule.RuleType) {
      case "SEQUENCE":
        // TODO: highlightify
        // requires change to how we're doing highlighting, since it's currently
        // span equality; this covers multiple spans
        return (
          <div>
            Sequence ({formatSpan(trace)})
            <ol style={{ marginTop: 0 }}>
              {trace.ItemTraces.map((itemTrace, idx) => (
                <li key={idx}>
                  <TraceNode
                    grammar={grammar}
                    trace={itemTrace}
                    {...highlightProps}
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
              {...highlightProps}
            />
          </div>
        );
      case "REF": {
        return (
          <div>
            {highlightWrapper(
              null,
              <RuleNameView
                id={grammar.TopLevelRules[rule.Ref]}
                name={rule.Ref}
                {...highlightProps}
              />,
            )}
            <br />
            <TraceNode
              grammar={grammar}
              trace={trace.RefTrace}
              {...highlightProps}
            />
          </div>
        );
      }
      case "KEYWORD":
        return highlightWrapper("rule-keyword", `"${rule.Keyword}"`);
      case "REGEX":
        return highlightWrapper(
          "rule-regex",
          `"${trace.RegexMatch.replace("\n", "\\n").replace("\t", "\\t")}"`,
        );
      case "SUCCEED":
        return <span className="rule-succeed">&lt;succeed&gt;</span>;
      default:
        console.error(trace);
        return <pre>{JSON.stringify(trace)}</pre>
    }
  }

}
