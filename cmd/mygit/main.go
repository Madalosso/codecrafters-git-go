package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	// "github.com/spf13/cobra" TODO: Use cobra to handle flags and commands
)

var ()

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

			_, err = writeTree(currentDir)

			// 2a. if file -> Create blob object and record its SHA hash
			// 2b. if dir -> Create tree object and record it. (recursive to handle nested dirs)
			// 3. Write the tree object to .git/objects dir

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

func writeFileFromPayload(payload []byte) ([20]byte, error) {

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

		if file.Name() == ".git" {
			continue
		}

		// Check if the file is a directory
		if file.IsDir() {
			// Recursively call writeTree
			// writeTree(fullPath)
			//
		} else {
			// Print the file name
			fmt.Println(file.Name())
			fmt.Println(fullFilePath)

			hash, _ := writeBlobObject(fullFilePath)

			treeEntries = append(treeEntries, TreeEntries{
				mode:       "100644", //TODO: Find file permission to properly set this
				objectType: "blob",   //TODO: Enum?
				name:       file.Name(),
				hash:       hash,
			})
		}
	}

	header := fmt.Sprintf("tree %d\000", len(treeEntries)) // wrong len(treeEntries)
	treePayload := []byte(header)
	for _, entry := range treeEntries {
		fmt.Println(entry.name)
		entryContent := []byte(entry.mode)
		// treePayload = append(treePayload, []byte("100644")...)
		entryContent = append(entryContent, []byte(entry.mode)...)
		entryContent = append(entryContent, []byte(entry.name)...)
		entryContent = append(entryContent, '\000') // sera?
		// entryContent = append(entryContent, 0) //  ou sera?
		entryContent = append(entryContent, entry.hash[:]...)
		fmt.Println(entryContent)
		treePayload = append(treePayload, entryContent...)
	}

	return writeFileFromPayload(treePayload)
}

func writeBlobObject(_filepath string) ([20]byte, error) {
	content, err := os.ReadFile(_filepath)
	if err != nil {
		// refactor to return the error (function signature)
		fmt.Fprintf(os.Stderr, "Error while reading file content: %s \n", err)
		os.Exit(1)
	}
	// get content length
	size := len(content)
	// create header
	header := fmt.Sprintf("blob %d\000", size)
	// create hash + write to file (if -w flag is present)
	blobPayload := append([]byte(header), content...)

	return writeFileFromPayload(blobPayload)
}
