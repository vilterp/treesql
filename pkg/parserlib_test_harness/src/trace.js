export function length(trace) {
  return trace.EndPos.Offset - trace.StartPos.Offset;
}

export function cursorIsWithin(trace) {
  return trace.CursorPos >= 0 && trace.CursorPos < length(trace);
}
