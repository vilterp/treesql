import React from "react";
import { formatSpan } from './span';
import classNames from "classnames";
import "./GrammarView.css";
import "./SourceView.css";

// Render a syntax-highlighted view of the source from the trace.
// Highlight hovered spans.

export class SourceView extends React.Component {
  render() {
    const {
      trace,
      grammar,
      highlightedSpan,
      onHighlightSpan,
    } = this.props;

    const formattedSpan = formatSpan(trace);
    const isHighlighted = formattedSpan === highlightedSpan;

    function highlightWrapper(element) {
      return (
        <span
          className={classNames("source-span", { highlighted: isHighlighted })}
          onMouseOver={() => { console.log(trace); onHighlightSpan(formattedSpan, true) }}
          onMouseOut={() => onHighlightSpan(formattedSpan, false)}
        >
          {element}
        </span>
      )
    }

    if (!trace) {
      return ""; // un-filled-in sequence items
    }

    const highlightProps = {
      onHighlightSpan: onHighlightSpan,
      highlightedSpan: highlightedSpan,
    };

    const rule = grammar.RulesByID[trace.RuleID];
    switch (rule.RuleType) {
      case "SEQUENCE":
        return (
          <span>
            {trace.ItemTraces.map((itemTrace, idx) => (
              <SourceView
                key={idx}
                trace={itemTrace}
                grammar={grammar}
                {...highlightProps}
              />
            ))}
          </span>
        );
      case "CHOICE":
        return (
          <SourceView
            trace={trace.ChoiceTrace}
            grammar={grammar}
            {...highlightProps}
          />
        );
      case "REF": {
        return (
          <SourceView
            trace={trace.RefTrace}
            grammar={grammar}
            {...highlightProps}
          />
        );
      }
      case "KEYWORD":
        return highlightWrapper(
          <span className="rule-keyword">{rule.Keyword}</span>
        );
      case "REGEX":
        return highlightWrapper(
          <span
            className="rule-regex"
            style={{ whiteSpace: "pre" }}
          >
            {trace.RegexMatch}
          </span>
        );
      case "SUCCEED":
        return null;
      default:
        console.error(trace);
        return <pre>{JSON.stringify(trace)}</pre>
    }
  }
}