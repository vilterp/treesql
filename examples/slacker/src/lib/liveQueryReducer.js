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
      return {
        tree: updateAtSelection(state.tree, action.queryPath, action.newRecord)
      }

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
    // replacing with NewRecord, which may have fields that weren't in the original selction...
    // would be good just to send the keys in your selection
    return {
      ...record,
      ...selection
    };
  } else {
    const fieldName = path[0].selection; // have to converge on terminology here lol
    return {
      ...record,
      [fieldName]: updateAtSelection(record[fieldName], path.slice(1), selection)
    }
  }
}
