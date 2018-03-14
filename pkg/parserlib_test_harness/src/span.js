export function formatPos(pos) {
  return `${pos.Line}:${pos.Col}`
}

export function formatSpan(trace) {
  return `${formatPos(trace.StartPos)} => ${formatPos(trace.EndPos)}`
}