package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
	// "github.com/spf13/cobra" TODO: Use cobra to handle flags and commands
)

const (
	authorName  = "Otavio Migliavacca Madalosso"
	authorEmail = "otaviomadalosso@gmail.com"
)

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Fprintf(os.Stderr, "Logs from your program will appear here!\n")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}
	command := os.Args[1]
	if command == "init" {
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")
	} else {
		// any command other than init should make sure the working dir is git valid
		checkInit()
		switch command {
		case "cat-file":
			// fmt.Println(os.Args)
			if len(os.Args) < 4 {
				fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <blobHash>\n")
				os.Exit(1)
			}
			// check expected args
			flag := os.Args[2]
			if flag != "-p" {
				fmt.Fprintf(os.Stderr, "missing mandatory flag -p: \n")
				os.Exit(1)
			}
			blobHash := os.Args[3]
			objectPath := hashToFilePath(blobHash)
			// read file content
			fileContent, err := os.ReadFile(objectPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading the file: %s \n", err)
				os.Exit(1)
			}
			decompressed, err := DecompressZlib(fileContent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while decompressing file content: %s \n", err)
				os.Exit(1)
			}
			// fmt.Println(decompressed)

			_, _, fileContent, err = ParseObjectContent(decompressed)
			// fileType, fileLength, fileContent, err := ParseObjectContent(decompressed)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while parsing file content: %s \n", err)
				os.Exit(1)
			}
			// fmt.Println(fileType, fileLength, string(fileContent))
			fmt.Print(string(fileContent))

		case "hash-object":

			// fmt.Println(os.Args)
			if len(os.Args) < 4 {
				fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w filepath\n")
				os.Exit(1)
			}
			// check expected args
			flag := os.Args[2]
			if flag != "-w" {
				fmt.Fprintf(os.Stderr, "missing mandatory flag -w: \n")
				os.Exit(1)
			}
			_filepath := os.Args[3]
			// read file content
			hash, _ := writeBlobObject(_filepath)

			// Refactor: function errors from writeBlobObject are
			// printing to stderr and exiting the program.
			// Instead, return the error and let the caller handle it.
			// if err != nil {
			// 	fmt.Fprintf(os.Stderr, "%s \n", err)
			// 	os.Exit(1)
			// }
			fmt.Printf("%x", hash)

		case "ls-tree":
			if len(os.Args) < 4 {
				fmt.Fprintf(os.Stderr, "usage: mygit ls-tree --name-only <tree_sha>\n")
				os.Exit(1)
			}
			flag := os.Args[2]
			if flag != "--name-only" {
				fmt.Fprintf(os.Stderr, "missing mandatory flag --name-only: \n")
				os.Exit(1)
			}
			// get tree sha
			treeSha := os.Args[3]

			// 1.search .git/objects for the treeSha entry
			treePath := hashToFilePath(treeSha)
			// handle err as well

			fileContent, err := os.ReadFile(treePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading the file: %s \n", err)
				os.Exit(1)
			}
			// 2.Zlib decompression
			fileContent, err = DecompressZlib(fileContent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while decompressing file content: %s \n", err)
				os.Exit(1)
			}

			// 3.Parse decompressed content (get tree entries)
			fileType, _, content, err := ParseObjectContent(fileContent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while parsing file content: %s \n", err)
				os.Exit(1)
			}
			if fileType != "tree" {
				fmt.Fprintf(os.Stderr, "tree_sha provided is not a tree object\n")
				os.Exit(1)
			}
			treeEntries, err := ParseTreeEntry(content)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while parsing tree entries: %s \n", err)
				os.Exit(1)
			}
			// 4.Print entries (name only, alphabetical order)
			sort.Slice(treeEntries, func(i, j int) bool {
				return treeEntries[i].name < treeEntries[j].name
			})
			for _, entry := range treeEntries {
				fmt.Println(entry.name)
				// TODO: full print
			}
		case "write-tree":
			// 1. Iterate over files/dirs within pwd (ignoring .git)
			currentDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %s \n", err)
				os.Exit(1)
			}

			hash, err := writeTree(currentDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while writing tree: %s \n", err)
				os.Exit(1)
			}
			fmt.Printf("%x", hash)

		case "commit-tree":
			// adopt Cobra to handle this
			// mygit commit-tree <tree_sha> -p <commit_sha> -m <message>
			// -p flag and value is optional
			var (
				treeSha   string
				parentSha string
				message   string
			)
			switch len(os.Args) {
			case 5: // assume without -p + commit_sha (mygit commit-tree <tree_sha> -m <message>)
				treeSha = os.Args[2]
				message = os.Args[4]
			case 7: // mygit commit-tree <tree_sha> -p <commit_sha> -m <message>
				treeSha = os.Args[2]
				parentSha = os.Args[4]
				message = os.Args[6]
			default:
				fmt.Fprintf(os.Stderr, "usage: mygit commit-tree <tree_sha> [-p <commit_sha>] -m <message>\n")
				os.Exit(1)
			}
			hash, err := buildCommitTree(treeSha, parentSha, message)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while writing commit tree: %s \n", err)
				os.Exit(1)
			}
			fmt.Printf("%x", hash)

		default:
			fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
			os.Exit(1)
		}
	}

}

func checkInit() {
	path := "./.git"
	fs, err := os.Stat(path)
	if os.IsNotExist(err) || !fs.IsDir() {
		panic("git not initialized (missing .git folder)")
	}
}

// TODO: test what if hash empty? return string, error?
func hashToFilePath(hash string) string {
	prefix := hash[:2]
	filepath := hash[2:]

	objectsDirPath := ".git/objects/"
	objectPath := fmt.Sprintf("%s%s/%s", objectsDirPath, prefix, filepath)
	return objectPath
}

func buildCommitTree(treeSha, parentSha, message string) ([20]byte, error) {
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

func writeTree(pathname string) ([20]byte, error) {
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
			hashTree, err := writeTree(fullFilePath)
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
			hash, _ := writeBlobObject(fullFilePath)
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

func writeBlobObject(_filepath string) ([20]byte, error) {
	content, err := os.ReadFile(_filepath)
	if err != nil {
		// refactor to return the error (function signature)
		fmt.Fprintf(os.Stderr, "Error while reading file content: %s \n", err)
		os.Exit(1)
	}
	return WriteFileFromPayload(content, "blob")
}
