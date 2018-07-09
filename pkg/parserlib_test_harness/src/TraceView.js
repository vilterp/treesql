import React from "react";
import { RuleNameView } from './RuleNameView';
import { formatSpan } from './span';
import classNames from "classnames";import { cursorIsWithin } from './trace';
import "./SourceView.css";
import "./TraceView.css";
import "./GrammarView.css"; // factor out the common parts? idk

export class TraceView extends React.Component {
  render() {
    return (
      <div className="trace-view">
        {this.props.error
          ? <p style={{ color: "red" }}>Error: {this.props.error}</p>
          : null}
        <TraceNode {...this.props} />
      </div>
    )
  }
}

class TraceNode extends React.Component {
  renderInternal(rule) {
    const {
      grammar,
      trace,
      onHighlightRule,
      highlightedRuleID,
      onHighlightSpan,
      highlightedSpan,
    } = this.props;

    const highlightProps = {
      onHighlightRule: onHighlightRule,
      highlightedRuleID: highlightedRuleID,
      onHighlightSpan: onHighlightSpan,
      highlightedSpan: highlightedSpan,
    };

    const formattedSpan = formatSpan(trace);
    const isHighlightedSpan = formattedSpan === highlightedSpan;

    function highlightSpanWrapper(className, element) {
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

    const isHighlightedRule = highlightedRuleID === trace.RuleID;

    function highlightRuleWrapper(element) {
      // TODO: DRY w/ GrammarView...?
      return (
        <span
          className={classNames("rule-def", { highlighted: isHighlightedRule })}
          onMouseOver={() => onHighlightRule(trace.RuleID, true)}
          onMouseOut={() => onHighlightRule(trace.RuleID, false)}
        >
          {element}
        </span>
      );
    }

    switch (rule.RuleType) {
      case "SEQUENCE":
        // TODO: highlightify
        // requires change to how we're doing highlighting, since it's currently
        // span equality; this covers multiple spans
        return (
          <div>
            {highlightRuleWrapper(
              <span>Sequence</span>
            )}
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
            {highlightRuleWrapper(
              <span>Choice {trace.ChoiceIdx}</span>
            )}
            <br/>
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
            {highlightSpanWrapper(
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
        return highlightRuleWrapper(
          highlightSpanWrapper("rule-keyword", `"${rule.Keyword}"`),
        );
      case "REGEX":
        return highlightRuleWrapper(
          highlightSpanWrapper(
            "rule-regex",
            `"${trace.RegexMatch.replace("\n", "\\n").replace("\t", "\\t")}"`,
          ),
        );
      case "MAP":
        return (
          <div>
            MAP
            <br />
            <TraceNode
              grammar={grammar}
              trace={trace.InnerTrace}
              {...highlightProps}
            />
          </div>
        );
      case "SUCCEED":
        return <span className="rule-succeed">&lt;succeed&gt;</span>;
      default:
        console.error(trace);
        return <pre>{JSON.stringify(trace)}</pre>
    }
  }

  render() {
    const {
      trace,
      grammar,
    } = this.props;

    if (!trace) {
      return <span>(empty)</span>;
    }

    const rule = grammar.RulesByID[trace.RuleID];
    const within = ["KEYWORD", "REGEX"].includes(rule.RuleType) && cursorIsWithin(trace);


    return (
      <div style={{ textDecoration: within ? "underline" : "none" }}>
        {this.renderInternal(rule, trace)}
      </div>
    );
  }

}
