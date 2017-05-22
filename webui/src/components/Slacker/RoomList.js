import React from 'react';

function RoomList({ rooms }) {
  return (
    <ul>
      {rooms.map((room) => (
        <li key={room.id}>
          {room.name}
        </li>
      ))}
    </ul>
  );
}

export default RoomList;
