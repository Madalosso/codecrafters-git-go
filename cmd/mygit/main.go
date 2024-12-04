package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
			// prefix := blobHash[:2]
			// filepath := blobHash[2:]
			// objectsDirPath := ".git/objects/"
			// objectPath := fmt.Sprintf("%s%s/%s", objectsDirPath, prefix, filepath)

			// read file content
			fileContent, err := os.ReadFile(objectPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading the file: %s \n", err)
				os.Exit(1)
			}
			decompressed, err := decompressZlib(fileContent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while decompressing file content: %s \n", err)
				os.Exit(1)
			}
			// fmt.Println(decompressed)

			_, _, fileContent, err = parseObjectContent(decompressed)
			// fileType, fileLength, fileContent, err := parseObjectContent(decompressed)
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
			content, err := os.ReadFile(_filepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading file content: %s \n", err)
				os.Exit(1)
			}
			// get content length
			size := len(content)
			// create header
			header := fmt.Sprintf("blob %d\000", size)
			// create hash + write to file (if -w flag is present)
			hashPayload := append([]byte(header), content...)
			h := sha1.New()

			// SHA hash input = header <(header = type(blob) + ' '(space) + <size>\0)> + uncompressed content
			h.Write(hashPayload)
			hash := h.Sum(nil)

			// print hash to stdout
			fmt.Printf("%x", hash)
			hashFilePath := hashToFilePath(fmt.Sprintf("%x", hash))
			// fmt.Println(hashFilePath)

			compressedFileContent, err := compressZlib(hashPayload)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while compressing file content: %s \n", err)
				os.Exit(1)
			}

			// Write dirs and file
			dirs := filepath.Dir(hashFilePath)
			err = os.MkdirAll(dirs, 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while creating directories: %s \n", err)
				os.Exit(1)
			}
			os.WriteFile(hashFilePath, compressedFileContent, 0644)

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

// TODO: Consider making the type an enum
// <type> <length>\0<data>
// blob 11\0hello world
func parseObjectContent(data []byte) (string, int, []byte, error) {
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

// maybe create a struct for zlib operations (compress/decompress+read+write)
func decompressZlib(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	reader, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

func compressZlib(data []byte) ([]byte, error) {
	var b bytes.Buffer
	writer := zlib.NewWriter(&b)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	writer.Close()
	return b.Bytes(), nil
}
