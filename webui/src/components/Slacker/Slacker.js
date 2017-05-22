import _ from 'lodash';
import React, { Component } from 'react';
import { connect } from 'react-redux';
import RoomList from './RoomList';
import Room from './Room';

// TODO: some kind of room read count
export const QUERY = `
  many rooms {
    id,
    name,
    messages: many messages {
      id,
      timestamp,
      body,
      user: one users {
        name
      }
    },
    room_users: many room_users {
      user: one users {
        id,
        name
      }
    }
  }
  live
`;

class Slacker extends Component {

  constructor() {
    super();
    this.selectRoom = this.selectRoom.bind(this);
  }

  // things should be returned keyed by id so this is unnecessary
  findRoom() {
    return _.find(this.props.queryTree, { id: this.state.currentRoomID });
  }

  componentWillMount() {
    this.setState({
      currentRoomID: null,
      currentUserID: null
    });
  }

  selectRoom(roomID) {
    // this should be a react router thing
    this.setState({
      currentRoomID: roomID
    });
  }

  render() {
    const { queryTree } = this.props;
    console.log('queryTree', queryTree);
    // ^v TODO: I guess make some sort of HOC that contains this
    if (!this.state.currentUserID) {
      return (
        <div className="signin">
          <h2>Who are you?</h2>
          <form>
            <input
              type="text"
              placeholder="User ID no."
              onInput={(evt) => { this.setState({ possibleUserID: evt.target.value }) }} />
            <button
              onClick={(evt) => {
                evt.preventDefault();
                this.setState({ currentUserID: this.state.possibleUserID });
              }}>
              Sign In
            </button>
          </form>
        </div>
      );
    } else {
      if (!queryTree) {
        return (
          <div className="loading">
            Loading Slacker...
          </div>
        );
      } else {
        return (
          <div>
            <h1>Slacker</h1>
            <div style={{ display: 'flex' }}>
              <RoomList
                rooms={queryTree}
                selectRoom={this.selectRoom}
                currentRoomID={this.state.currentRoomID} />
              {this.state.currentRoomID
                ? <Room
                    room={this.findRoom(this.state.currentRoomID)}
                    currentUserID={this.state.currentUserID} />
                : <div className="select-a-room">Select a room</div>}
            </div>
          </div>
        );
      }
    }
  }

}

function mapStateToProps(state) {
  return {
    queryTree: state.slacker.tree
  };
}

function mapDispatchToProps(dispatch) {
  return {}; // update state... send statements...
}

export default connect(mapStateToProps, mapDispatchToProps)(Slacker);
