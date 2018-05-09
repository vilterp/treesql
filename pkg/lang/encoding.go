package lang

import (
	"bufio"
	"bytes"
)

func Encode(v Value) (string, error) {
	sb := bytes.NewBufferString("")
	w := bufio.NewWriter(sb)
	// hmm, have to figure out a way to get rid
	// of iterators and stuff that need a caller
	if err := v.WriteAsJSON(w, nil); err != nil {
		return "", err
	}
	w.Flush()
	return sb.String(), nil
}
