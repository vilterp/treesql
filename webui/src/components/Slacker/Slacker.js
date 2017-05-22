import _ from 'lodash';
import React, { Component } from 'react';
import { connect } from 'react-redux';
import RoomList from './RoomList';
import MessageList from './MessageList';

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
`;

class Slacker extends Component {

  messagesForRoom() {
    return _.find(this.props.queryState, { id: this.state.currentRoomID }).messages;
  }

  render() {
    const { queryState } = this.props;
    // ^v TODO: I guess make some sort of HOC that contains this
    if (_.isEqual(queryState, {})) {
      return (
        <div className="loading">
          Loading Slacker...
        </div>
      );
    } else {
      return (
        <div>
          <RoomList rooms={queryState} />
          {this.state.currentRoomID
            ? <MessageList messages={this.messagesForRoom(this.state.currentRoomID)} />
            : <div className="select-a-room">Select a room</div>}
        </div>
      );
    }
  }

}

function mapStateToProps(state) {
  return {
    queryState: state.slacker
  };
}

function mapDispatchToProps(dispatch) {
  return {}; // update state... send statements...
}

export default connect(mapStateToProps, mapDispatchToProps)(Slacker);
