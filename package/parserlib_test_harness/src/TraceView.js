import React from "react";

function renderPos(pos) {
  return `${pos.Line}:${pos.Col}`
}

function renderSpan(trace) {
  return `${renderPos(trace.StartPos)} => ${renderPos(trace.EndPos)}`
}

export class TraceView extends React.Component {
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
                  <TraceView grammar={grammar} trace={itemTrace} />
                </li>
              ))}
            </ol>
          </div>
        );
      case "CHOICE":
        return (
          <div>
            Choice {trace.ChoiceIdx}<br/>
            <TraceView grammar={grammar} trace={trace.ChoiceTrace} />
          </div>
        );
      case "REF": {
        return (
          <div>
            Ref: {rule.Ref}<br />
            <TraceView grammar={grammar} trace={trace.RefTrace} />
          </div>
        );
      }
      case "KEYWORD":
        return <span>Keyword "{rule.Keyword}"</span>;
      case "REGEX":
        return <span>Regex "{trace.RegexMatch}"</span>;
      case "SUCCEED":
        return <span>Succeed</span>;
      default:
        console.error(trace);
        return <pre>{JSON.stringify(trace)}</pre>
    }
  }

}
