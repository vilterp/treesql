import React, { Component } from 'react';
import "./App.css";
import { TraceView } from './TraceView';
import { GrammarView } from './GrammarView';
import { SourceView } from './SourceView';

const INITIAL_QUERY = `MANY blog_posts {
  id,
  title,
  comments: MANY comments {
    id,
    body
  }
}`;

// TODO: bundle all these "onhighlightX" and "highlightedX" props to a
// highlightContext or something

class App extends Component {
  constructor() {
    super();
    this.state = {
      query: INITIAL_QUERY,
      grammar: null,
      trace: null,

      highlightedRuleID: null,
      highlightedSpan: null,
    }
  }

  componentDidMount() {
    this.fetchGrammar();
    this.fetchQuery(INITIAL_QUERY);
  }

  fetchGrammar() {
    fetch("http://localhost:9999/grammar").then((resp) => {
      resp.json().then((grammar) => {
        console.log("grammar:", grammar);
        this.setState({
          grammar: grammar,
        })
      }).catch((err) => {
        console.error("error parsing grammar:", err);
      });
    });
  }

  fetchQuery(query) {
    const completionReq = {
      Input: query,
      CursorPos: 0,
    };
    fetch("http://localhost:9999/completions", {
      method: "POST",
      body: JSON.stringify(completionReq),
    }).then((resp) => {
      resp.json().then((completionResp) => {
        console.log("trace:", completionResp);
        this.setState({
          trace: completionResp,
        });
      }).catch((err) => {
        console.error("error parsing completions:", err);
      });
    });
  }

  handleQueryUpdate(val) {
    this.setState({
      query: val,
    });
    this.fetchQuery(val);
  }

  handleHighlightRule = (ruleID, highlight) => {
    this.setState({
      highlightedRuleID: highlight ? ruleID : null,
    });
  }

  handleHighlightSpan = (span, highlight) => {
    this.setState({
      highlightedSpan: highlight ? span : null,
    });
  }

  render() {
    return (
      <div className="App">
        <h1>TreeSQL Parser Test Harness</h1>
        <div className="app-container">
          <div className="grid-cell app-editor">
            <textarea
              style={{ fontFamily: "monospace" }}
              cols={80}
              rows={10}
              value={this.state.query}
              onChange={(evt) => this.handleQueryUpdate(evt.target.value)}
            />
          </div>
          <div className="grid-cell app-sourceview">
            {this.state.trace && this.state.grammar
              ? <SourceView
                trace={this.state.trace.Trace}
                grammar={this.state.grammar}
                onHighlightSpan={this.handleHighlightSpan}
                highlightedSpan={this.state.highlightedSpan}
              />
              : <span>&lt;don't have both trace & grammar yet&gt;</span>}
          </div>
          <div className="grid-cell app-traceview">
            <h3>Trace</h3>
            {this.state.trace && this.state.grammar
              ? <TraceView
                trace={this.state.trace.Trace}
                grammar={this.state.grammar}
                onHighlightRule={this.handleHighlightRule}
                highlightedRuleID={this.state.highlightedRuleID}
                onHighlightSpan={this.handleHighlightSpan}
                highlightedSpan={this.state.highlightedSpan}
              />
              : <span>&lt;don't have both trace & grammar yet&gt;</span>}
          </div>
          <div className="grid-cell app-grammar">
            <h3>Grammar</h3>
            {this.state.grammar
              ? <GrammarView
                grammar={this.state.grammar}
                onHighlightRule={this.handleHighlightRule}
                highlightedRuleID={this.state.highlightedRuleID}
              />
              : <span>&lt;don't have grammar yet&gt;</span>}
          </div>
        </div>
      </div>
    );
  }
}

export default App;
