import {
  INITIAL_RESULT,
  RECORD_UPDATE,
  TABLE_UPDATE
} from './liveQueryActions';

const initialState = {};

export default function update(state = initialState, action) {
  switch (action.type) {
    case INITIAL_RESULT: {
      const withSchema = ensureSchema(state, action.Schema);
      return setResultAtPath(withSchema, [], action.Data)
    }
    case TABLE_UPDATE:
      console.log('TODO: table update');
      return state;
    
    case RECORD_UPDATE:
      console.log('TODO: record update');
      return state;

    default:
      return state;
  }
}

function ensureSchema(state, schema) {
  console.log('TODO: ensure schema');
  return state;
}

function setResultAtPath(state, path, data) {
  console.log('TODO: set result at path');
  return state;
}
