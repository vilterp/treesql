package util

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// From https://gist.github.com/turtlemonvh/e4f7404e28387fadb8ad275a99596f67
func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("error parsing string 1: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("error parsing string 2: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}
