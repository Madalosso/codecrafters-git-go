package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
)

func WriteFileFromPayload(payload []byte) ([20]byte, error) {

	h := sha1.New()
	h.Write(payload)
	payloadHash := h.Sum(nil)
	hashFilePath := hashToFilePath(fmt.Sprintf("%x", payloadHash))
	compressedFileContent, err := CompressZlib(payload)
	if err != nil {
		// refactor to return the error (function signature)
		fmt.Fprintf(os.Stderr, "Error while compressing file content: %s \n", err)
		os.Exit(1)
	}
	dirs := filepath.Dir(hashFilePath)
	err = os.MkdirAll(dirs, 0755)
	if err != nil {
		// refactor to return the error (function signature)
		fmt.Fprintf(os.Stderr, "Error while creating directories: %s \n", err)
		os.Exit(1)
	}
	os.WriteFile(hashFilePath, compressedFileContent, 0644)

	var hashArray [20]byte
	copy(hashArray[:], payloadHash)
	return hashArray, nil
}
