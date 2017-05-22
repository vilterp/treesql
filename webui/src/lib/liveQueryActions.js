export const INITIAL_RESULT = 'INITIAL_RESULT';
export const initialResult = (schema, data) => ({
  type: INITIAL_RESULT,
  schema,
  data
});

export const TABLE_UPDATE = 'TABLE_UPDATE';
export const tableUpdate = (queryPath, selection) => ({
  type: TABLE_UPDATE,
  queryPath,
  selection
});

export const RECORD_UPDATE = 'RECORD_UPDATE';
export const recordUpdate = (queryPath, oldRecord, newRecord) => ({
  type: RECORD_UPDATE,
  oldRecord,
  newRecord
});

// idk, maybe this should be in TreeSQLClient.js
export function updateToAction(update) {
  const payload = update.payload;
  switch (update.type) {
    case 'initial_result':
      return initialResult(payload.Schema, payload.Data);

    case 'record_update':
      return recordUpdate(payload.QueryPath, payload.TableEvent.OldRecord, payload.TableEvent.NewRecord);

    case 'table_update':
      return tableUpdate(payload.QueryPath, payload.Selection);
    
    default:
      console.warn('unhandled message from live query:', update);
  }
}
