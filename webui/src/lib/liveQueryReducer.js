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
      return {
        tree: updateAtSelection(state.tree, action.queryPath, action.selection[0])
      };

    case RECORD_UPDATE:
      console.log('TODO: record update');
      return state;

    default:
      return state;
  }
}

function updateAtSelection(records, path, selection) {
  if (path.length === 0) {
    return [
      ...records,
      selection
    ];
  } else {
    const idComponent = path[0];
    const id = idComponent.id;
    return records.map((record) => (
      // sooo... this requires:
      // a) your table's primary key col is "id"
      // b) the PK col is in the live query at each level
      // reasonable requirement, but it should be documented
      record.id === id
      ? updateAtRecord(record, path.slice(1), selection)
      : record
    ));
  }
}

function updateAtRecord(record, path, selection) {
  if (path.length === 0) {
    console.log('idk what to do here', record, path, selection);
    return record;
  } else {
    console.log('idk what to do here either', record, path, selection);
    return record;
  }
}
