import _ from 'lodash';

const HISTORY_IDX_KEY = "next_statement_history_idx";
const HISTORY_ITEM_PREFIX = 'statement_history_item'

export function storeStatement(statement) {
  const nextHistoryIdx = JSON.parse(localStorage.getItem(HISTORY_IDX_KEY)) || 0;
  localStorage.setItem(HISTORY_IDX_KEY, JSON.stringify(nextHistoryIdx+1));
  localStorage.setItem(`${HISTORY_ITEM_PREFIX}/${nextHistoryIdx}`, JSON.stringify(statement))
}

export function getStatementHistory() {
  const nextHistoryIdx = JSON.parse(localStorage.getItem(HISTORY_IDX_KEY)) || 0;
  const history = [];
  for (var i = 0; i < nextHistoryIdx; i++) {
    history.push(JSON.parse(localStorage.getItem(`${HISTORY_ITEM_PREFIX}/${i}`)));
  }
  return _.uniq(history);
}
