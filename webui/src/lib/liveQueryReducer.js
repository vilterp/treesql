import _ from 'lodash';
import {
  INITIAL_RESULT,
  RECORD_UPDATE,
  TABLE_UPDATE
} from './liveQueryActions';

const initialState = {
  tree: null
};

export default function update(state = initialState, action) {
  switch (action.type) {
    case INITIAL_RESULT: {
      return {
        tree: action.data
      };
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

function setResultAtPath(state, path, data) {
  console.log('TODO: set result at path');
  return state;
}
