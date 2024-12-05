package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
)

func WriteFileFromPayload(payload []byte, objectType string) ([20]byte, error) {
	header := []byte(fmt.Sprintf("%s %d\000", objectType, len(payload)))
	completePayload := append(header, payload...)
	h := sha1.New()
	h.Write(completePayload)
	payloadHash := h.Sum(nil)
	hashFilePath := HashToFilePath(fmt.Sprintf("%x", payloadHash))
	compressedFileContent, err := CompressZlib(completePayload)
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

func HashToFilePath(hash string) string {
	prefix := hash[:2]
	filepath := hash[2:]

	objectsDirPath := ".git/objects/"
	objectPath := fmt.Sprintf("%s%s/%s", objectsDirPath, prefix, filepath)
	return objectPath
}
