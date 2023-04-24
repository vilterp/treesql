package lang

type Iterator interface {
	// Next returns the next value, or an error if we have reached the
	// end of the sequence.
	Next(caller Caller) (Value, error)
	Close() error
}

// Map iterator

type mapIterator struct {
	innerIterator Iterator
	f             vFunction
}

var _ Iterator = &mapIterator{}

func (mi *mapIterator) Next(c Caller) (Value, error) {
	next, err := mi.innerIterator.Next(c)
	if err != nil {
		// TODO: close inner iterator? idk
		return nil, err
	}
	val, err := c.Call(mi.f, []Value{next})
	return val, err
}

func (mi *mapIterator) Close() error {
	return mi.innerIterator.Close()
}

// Filter iterator

type filterIterator struct {
	innerIterator Iterator
	f             vFunction
}

func (fi *filterIterator) Next(c Caller) (Value, error) {
	for {
		// Get the next value.
		next, err := fi.innerIterator.Next(c)
		var isEOE bool
		switch err.(type) {
		case *endOfIteration:
			isEOE = true
		default:
			if err != nil {
				return nil, err
			}
		}
		// Check for end of iteration.
		if isEOE {
			return nil, EndOfIteration
		}
		// Call the func.
		res, err := c.Call(fi.f, []Value{next})
		if err != nil {
			return nil, err
		}
		// Return the val if true.
		if *mustBeVBool(res) {
			return next, nil
		}
	}
}

func (fi *filterIterator) Close() error {
	return fi.innerIterator.Close()
}

// Array iterator

type arrayIterator struct {
	pos  int
	vals []Value
}

var _ Iterator = &arrayIterator{}

type endOfIteration struct{}

var EndOfIteration = &endOfIteration{}

func (endOfIteration) Error() string {
	return "reached end of iterator"
}

func NewArrayIterator(vals []Value) *arrayIterator {
	return &arrayIterator{
		pos:  0,
		vals: vals,
	}
}

func (ai *arrayIterator) Next(_ Caller) (Value, error) {
	if ai.pos == len(ai.vals) {
		return nil, EndOfIteration
	}
	val := ai.vals[ai.pos]
	ai.pos++
	return val, nil
}

func (ai *arrayIterator) Close() error {
	return nil
}

// TODO: mapIterator, filterIterator
// TODO: limitIterator, orderByIterator, offsetIterator
// TODO: aggregation iterators

// TODO: index iterator
// these should both push stack frames so record listeners can be installed
