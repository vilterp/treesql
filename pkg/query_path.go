package treesql

import (
	"fmt"
)

// queryPath is a linked list type deal
type queryPath struct {
	// only one of these should be not nil (ugh)
	Selection       *string
	ID              *string
	PreviousSegment *queryPath // up the tree
}

func (qp *queryPath) String() string {
	return fmt.Sprintf("%v", qp.flatten())
}

func (qp *queryPath) length() int {
	currentSegment := qp
	length := 0
	for currentSegment != nil {
		currentSegment = currentSegment.PreviousSegment
		length++
	}
	return length
}

type flattenedQueryPath = []map[string]string

func (qp *queryPath) flatten() flattenedQueryPath {
	length := qp.length()
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
