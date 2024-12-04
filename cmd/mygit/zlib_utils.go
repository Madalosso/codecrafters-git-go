package main

import (
	"bytes"
	"compress/zlib"
	"io"
)

func DecompressZlib(data []byte) ([]byte, error) {
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

func CompressZlib(data []byte) ([]byte, error) {
	var b bytes.Buffer
	writer := zlib.NewWriter(&b)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	writer.Close()
	return b.Bytes(), nil
}
