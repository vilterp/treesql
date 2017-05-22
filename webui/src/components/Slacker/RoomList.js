import React from 'react';
import classNames from 'classnames';
import './RoomList.css';

function RoomList({ rooms, currentRoomID, selectRoom }) {
  return (
    <div>
      <h2>Rooms</h2>
      <ul>
        {rooms.map((room) => (
          <li
            key={room.id}
            className={classNames('room', { 'current-room': currentRoomID === room.id })}
            onClick={() => selectRoom(room.id)}>
            #{room.name}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default RoomList;
