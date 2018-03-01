import React, { Component } from 'react';
import "./App.css";

const INITIAL_QUERY = `MANY blog_posts {
  id,
  title,
  comments: MANY comments {
    id,
    body
  }
}`;

class App extends Component {
  constructor() {
    super();
    this.state = {
      query: INITIAL_QUERY,
      grammar: null,
      trace: null,
    }
  }

  componentDidMount() {
    this.fetchGrammar();
    this.fetchQuery(INITIAL_QUERY);
  }

  fetchGrammar() {
    fetch("http://localhost:9999/grammar").then((resp) => {
      resp.json().then((grammar) => {
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

  render() {
    return (
      <div className="App">
        <h1>TreeSQL Test Harness</h1>
        <table>
          <tbody>
            <tr>
              <td>
              <textarea
                cols={80}
                rows={10}
                value={this.state.query}
                onChange={(evt) => this.handleQueryUpdate(evt.target.value)}
              />
              </td>
              <td>
                <pre>{this.state.query}</pre>
              </td>
            </tr>
            <tr style={{ verticalAlign: "top" }}>
              <td>
                <h3>Trace</h3>
                <pre>{JSON.stringify(this.state.trace, null, 2)}</pre>
              </td>
              <td>
                <h3>Grammar</h3>
                <pre>{JSON.stringify(this.state.grammar, null, 2)}</pre>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    );
  }
}

export default App;
