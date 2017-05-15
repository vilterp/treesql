package treesql

import (
	"encoding/json"
)

// QueryPath is a linked list type deal
type QueryPath struct {
	// only one of these should be not nil (ugh)
	Selection       *string
	ID              *string
	PreviousSegment *QueryPath // up the tree
}

func (qp *QueryPath) MarshalJSON() ([]byte, error) {
	return json.Marshal(qp.Flatten())
}

func (qp *QueryPath) Length() int {
	currentSegment := qp
	length := 0
	for currentSegment != nil {
		currentSegment = currentSegment.PreviousSegment
		length++
	}
	return length
}

func (qp *QueryPath) Flatten() []map[string]*string {
	array := make([]map[string]*string, qp.Length())
	currentSegment := qp
	for i := 0; currentSegment != nil; i++ {
		array[i] = map[string]*string{
			"selection": currentSegment.Selection,
			"id":        currentSegment.ID,
		}
		currentSegment = currentSegment.PreviousSegment
	}
	return array
}
