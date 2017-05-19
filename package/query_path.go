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

// idk is there an interface for this
func (qp *QueryPath) ToString() string {
	jsonRepr, _ := qp.MarshalJSON()
	return string(jsonRepr)
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

func (qp *QueryPath) Flatten() []map[string]string {
	length := qp.Length()
	array := make([]map[string]string, length)
	currentSegment := qp
	for i := 0; currentSegment != nil; i++ {
		pathSegment := map[string]string{}
		if currentSegment.Selection != nil {
			pathSegment["selection"] = *currentSegment.Selection
		}
		if currentSegment.ID != nil {
			pathSegment["id"] = *currentSegment.ID
		}
		array[length-i-1] = pathSegment
		currentSegment = currentSegment.PreviousSegment
	}
	return array
}
