package main

import (
	"bytes"
	"fmt"
	"strconv"
)

type TreeEntries struct {
	mode string
	name string
	hash [20]byte
}
type objectType = string

// TODO: Consider making the type an enum
// <type> <length>\0<data>
// blob 11\0hello world
func ParseObjectContent(data []byte) (objectType, int, []byte, error) {
	indexZeroByte := bytes.IndexByte(data, 0)
	if indexZeroByte == -1 {
		return "", 0, nil, fmt.Errorf("byte zero not found")
	}

	// <type> <length>
	// splits on space
	parts := bytes.Fields(data[:indexZeroByte])

	if len(parts) != 2 {
		return "", 0, nil, fmt.Errorf("header does not contain exactly two parts")
	}

	fileType := string(parts[0])
	contentLength, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return "", 0, nil, fmt.Errorf("invalid length: %v", err)
	}

	// <data>
	content := data[indexZeroByte+1:]
	// fmt.Println(content)
	if contentLength > len(content) {
		return "", 0, nil, fmt.Errorf("data length beyond slice boundary")
	}

	return fileType, contentLength, content, nil

}

func ParseTreeEntry(data []byte) ([]TreeEntries, error) {
	var entries []TreeEntries
	for len(data) > 0 {
		indexZeroByte := bytes.IndexByte(data, 0)
		if indexZeroByte == -1 {
			return nil, fmt.Errorf("byte zero not found")
		}

		// <mode> <name>\0<hash>
		// splits on space
		parts := bytes.Fields(data[:indexZeroByte])

		if len(parts) != 2 {
			return nil, fmt.Errorf("header does not contain exactly two parts")
		}

		mode := string(parts[0])
		name := string(parts[1])

		// <hash>
		var hash [20]byte
		copy(hash[:], data[indexZeroByte+1:indexZeroByte+21])

		entries = append(entries, TreeEntries{
			mode: mode,
			name: name,
			hash: hash,
		})

		// TODO: Check if this is valid, how this operation performs under the hood.
		// is this just a pointer moving forward?
		data = data[indexZeroByte+21:]
	}
	return entries, nil
}
