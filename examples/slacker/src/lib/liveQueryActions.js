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
  queryPath,
  oldRecord,
  newRecord
});

// idk, maybe this should be in TreeSQLClient.js
export function updateToAction(update) {
  switch (update.type) {
    case 'initial_result':
      return initialResult(update.initial_result.Schema, update.initial_result.Data);

    case 'record_update':
      return recordUpdate(
        update.record_update.QueryPath,
        update.record_update.TableEvent.OldRecord,
        update.record_update.TableEvent.NewRecord
      );

    case 'table_update':
      // TODO: this should come through as an empty list
      return tableUpdate(
        update.table_update.QueryPath || [],
        update.table_update.Selection
      );
    
    default:
      console.warn('unhandled message from live query:', update);
  }
}
