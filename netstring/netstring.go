// This package implements just enough [netstrings]
// for our homebrew IPC solution.
//
// [netstrings]: https://cr.yp.to/proto/netstrings.txt
package netstring

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func Encode(s string) string {
	return fmt.Sprintf("%d:%s,", len(s), s)
}

// Encode multiple strings into a nested netstring
func EncodeN(strings ...string) (ns string) {
	for _, s := range strings {
		ns += Encode(s)
	}
	return Encode(ns)
}

// A SplitFunc to be used in a bufio.Scanner
func SplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	colonIndex := bytes.IndexRune(data, ':')
	if colonIndex == -1 {
		// Haven't fully read the length part yet => skip:
		return 0, nil, nil
	}

	length, err := strconv.Atoi(string(data[:colonIndex]))
	if err != nil {
		return 0, nil, fmt.Errorf("split netstring: %w", err)
	}

	rest := data[colonIndex+1:]
	if len(rest) < length+1 { // +1 for "," at the end
		// Haven't read the whole netstring yet => skip:
		return 0, nil, nil
	}

	// The whole netstring should now be within the buffer
	return colonIndex + 1 + length + 1, rest[:length], nil
}

// Decode multiple concatenated netstrings into plain strings.
// This is NOT a reverse EncodeN() - the input here is not a nested
// netstring.
func DecodeMultiple(nstrings string) (results []string) {
	r := strings.NewReader(nstrings)
	s := bufio.NewScanner(r)
	s.Split(SplitFunc)
	for s.Scan() {
		results = append(results, s.Text())
	}
	return results
}
