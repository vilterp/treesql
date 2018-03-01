import React from "react";
import "./GrammarView.css";

// Render a syntax-highlighted view of the source from the trace.
// Highlight hovered spans.

export class SourceView extends React.Component {
  render() {
    const { trace, grammar } = this.props;

    if (!trace) {
      return ""; // un-filled-in sequence items
    }

    const rule = grammar.RulesByID[trace.RuleID];
    switch (rule.RuleType) {
      case "SEQUENCE":
        return (
          <span>
            {trace.ItemTraces.map((itemTrace) => (
              <SourceView trace={itemTrace} grammar={grammar} />
            ))}
          </span>
        );
      case "CHOICE":
        return <SourceView trace={trace.ChoiceTrace} grammar={grammar} />;
      case "REF": {
        return <SourceView trace={trace.RefTrace} grammar={grammar} />;
      }
      case "KEYWORD":
        return <span className="rule-keyword">{rule.Keyword}</span>
      case "REGEX":
        return (
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