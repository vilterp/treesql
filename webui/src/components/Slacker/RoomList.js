import React from 'react';

function RoomList({ rooms }) {
  return (
    <div>
      <h2>Rooms</h2>
      <ul>
        {rooms.map((room) => (
          <li key={room.id}>
            {room.name}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default RoomList;
