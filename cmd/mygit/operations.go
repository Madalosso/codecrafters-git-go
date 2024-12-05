package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func WriteTree(pathname string) ([20]byte, error) {
	// Open the directory
	dir, err := os.Open(pathname)
	if err != nil {
		// return "", err
		fmt.Fprintf(os.Stderr, "Error while opening directory: %s \n", err)
		os.Exit(1)
	}
	defer dir.Close()

	// Get the list of files
	fileInfo, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while reading directory files: %s \n", err)
		os.Exit(1)
	}

	treeEntries := []TreeEntries{}
	// Iterate over the files
	for _, file := range fileInfo {
		// Construct the full path
		fullFilePath := filepath.Join(pathname, file.Name())

		// skip .git dir
		if file.Name() == ".git" {
			continue
		}

		if file.IsDir() {
			// recursively create tree for sub directories
			hashTree, err := WriteTree(fullFilePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while writing tree: %s \n", err)
				os.Exit(1)
			}
			treeEntries = append(treeEntries, TreeEntries{
				mode:       "40000", //TODO: Find file permission to properly set this
				objectType: "tree",  //TODO: Enum?
				name:       file.Name(),
				hash:       hashTree,
			})
		} else {
			hash, _ := WriteBlobObject(fullFilePath)
			treeEntries = append(treeEntries, TreeEntries{
				mode:       "100644", //TODO: Find file permission to properly set this
				objectType: "blob",   //TODO: Enum?
				name:       file.Name(),
				hash:       hash,
			})
		}
	}

	// sort tree entries by name
	sort.Slice(treeEntries, func(i, j int) bool {
		return treeEntries[i].name < treeEntries[j].name
	})

	treePayload := []byte{}
	for _, entry := range treeEntries {
		entryContent := []byte(fmt.Sprintf("%s %s\000", entry.mode, entry.name))
		entryContent = append(entryContent, entry.hash[:]...)
		treePayload = append(treePayload, entryContent...)
	}
	return WriteFileFromPayload(treePayload, "tree")
}

func WriteBlobObject(_filepath string) ([20]byte, error) {
	content, err := os.ReadFile(_filepath)
	if err != nil {
		// refactor to return the error (function signature)
		fmt.Fprintf(os.Stderr, "Error while reading file content: %s \n", err)
		os.Exit(1)
	}
	return WriteFileFromPayload(content, "blob")
}

func BuildCommitTree(treeSha, parentSha, message string) ([20]byte, error) {
	now := time.Now()
	unixNow := now.Unix()
	_, offset := now.Zone()
	offsetHours := offset / 3600
	nowFormatted := fmt.Sprintf("%d %02d00", unixNow, offsetHours)

	// Content of the commit object
	content := []byte(fmt.Sprintf("tree %s\n", treeSha))
	if parentSha != "" {
		// Improve: Check consider possibility of multiple parents?
		content = append(content, []byte(fmt.Sprintf("parent %s\n", parentSha))...)
	}

	content = append(content, []byte(fmt.Sprintf("author %s <%s> %s\n", authorName, authorEmail, nowFormatted))...)
	content = append(content, []byte(fmt.Sprintf("commiter %s <%s> %s\n\n", authorName, authorEmail, nowFormatted))...)
	content = append(content, []byte(fmt.Sprintln(message))...)

	return WriteFileFromPayload(content, "commit")
}
