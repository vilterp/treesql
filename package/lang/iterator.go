package lang

type Iterator interface {
	// Next returns the next value, or an error if we have reached the
	// end of the sequence.
	Next() (Value, error)
	Close() error
}

type ArrayIterator struct {
	pos  int
	vals []Value
}

var _ Iterator = &ArrayIterator{}

type endOfIteration struct{}

var EndOfIteration = &endOfIteration{}

func (endOfIteration) Error() string {
	return "reached end of iterator"
}

// Array Iterator

func NewArrayIterator(vals []Value) *ArrayIterator {
	return &ArrayIterator{
		pos:  0,
		vals: vals,
	}
}

func (ai *ArrayIterator) Next() (Value, error) {
	if ai.pos == len(ai.vals) {
		return nil, EndOfIteration
	}
	val := ai.vals[ai.pos]
	ai.pos++
	return val, nil
}

func (ai *ArrayIterator) Close() error {
	return nil
}

// TODO: mapIterator, filterIterator
// also aggregation iterators

// TODO: table scan iterator
// TODO: index iterator
// these should both push stack frames so record listeners can be installed
