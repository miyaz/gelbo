package main

import (
	"fmt"
	"net/http"
	"strconv"

	_ "embed"
)

var (
	//go:embed websocket.html
	chatHTML string

	//go:embed grpc/proto/gelbo.proto
	gelboProto string
)

type File struct {
	Type    string
	Content string
}

func filesDLHandler(w http.ResponseWriter, r *http.Request) {
	filesMap := map[string]File{}
	filesMap["/chat/"] = File{Type: "text/html; charset=utf-8", Content: chatHTML}
	filesMap["/files/gelbo.proto"] = File{Type: "application/proto; charset=utf-8", Content: gelboProto}

	file, ok := filesMap[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		setStatusForLogger(http.StatusNotFound, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		setStatusForLogger(http.StatusMethodNotAllowed, r)
		return
	}
	w.Header().Set("Content-Type", file.Type)
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	fmt.Fprint(w, file.Content)

	setRespSizeForLogger(int64(len(file.Content)), r)
	setStatusForLogger(http.StatusOK, r)
}
