// this code implements the h2c part of HTTP/2.
//
// The h2c protocol is the non-TLS secured version of HTTP/2 which is not
// available from net/http.
//
// refs:
//   * https://github.com/veqryn/h2c
//   * https://cs.opensource.google/go/x/net/+/internal-branch.go1.19-vendor:http2/h2c/h2c.go
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
)

// Confirm implementation of http.Handler interface
var _ http.Handler = &HandlerH2C{}

// HandlerH2C implements http.Handler and enables h2c.
// Users who want h2c just need to provide a http.Handler to wrap, and an http2.Server.
// 	Example:
//
// 	router := http.NewServeMux()
//
// 	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		fmt.Fprint(w, "Hello World")
// 	})
//
// 	h2cWrapper := &h2c.HandlerH2C{
// 		Handler:  router,
// 		H2Server: &http2.Server{},
// 	}
//
// 	srv := http.Server{
// 		Addr:    ":8080",
// 		Handler: h2cWrapper,
// 	}
//
// 	srv.ListenAndServe()
type HandlerH2C struct {
	Handler  http.Handler
	H2Server *http2.Server
}

// ServeHTTP will serve with an HTTP/2 connection if possible using the `H2Server`.
// The request will be handled by the wrapped `Handler` in any case.
func (h *HandlerH2C) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// HTTP/2 With Prior Knowledge
	if r.Method == "PRI" && r.URL.Path == "*" && r.ProtoMajor == 2 {
		ctx := context.WithValue(r.Context(), "proto", "h2c")
		conn, err := initH2CWithPriorKnowledge(w)
		if err != nil {
			log.Printf("Error h2c with prior knowledge: %v", err)
			return
		}
		defer conn.Close()
		h.H2Server.ServeConn(conn, &http2.ServeConnOpts{
			Context: ctx,
			Handler: h.Handler,
		})
		return
	}

	ctx := context.WithValue(r.Context(), "proto", "http")
	r = r.WithContext(ctx)

	h.Handler.ServeHTTP(w, r)
}

// initH2CWithPriorKnowledge implements creating a h2c connection with prior
// knowledge (Section 3.4) and creates a net.Conn suitable for http2.ServeConn.
// All we have to do is look for the client preface that is suppose to be part
// of the body, and reforward the client preface on the net.Conn this function
// creates.
func initH2CWithPriorKnowledge(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijack not supported")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack failed: %v", err)
	}

	expectedBody := "SM\r\n\r\n"

	buf := make([]byte, len(expectedBody))
	n, err := io.ReadFull(rw, buf)
	if err != nil {
		return nil, fmt.Errorf("fail to read body: %v", err)
	}

	if bytes.Equal(buf[0:n], []byte(expectedBody)) {
		c := &rwConn{
			Conn:      conn,
			Reader:    io.MultiReader(bytes.NewBuffer([]byte(http2.ClientPreface)), rw),
			BufWriter: rw.Writer,
		}
		return c, nil
	}

	conn.Close()
	// log.Printf("Missing the request body portion of the client preface. Wanted: %v Got: %v", []byte(expectedBody), buf[0:n])
	return nil, errors.New("invalid client preface")
}

// bufWriter is a Writer interface that also has a Flush method.
type bufWriter interface {
	io.Writer
	Flush() error
}

// rwConn implements net.Conn but overrides Read and Write so that reads and
// writes are forwarded to the provided io.Reader and bufWriter.
type rwConn struct {
	net.Conn
	io.Reader
	BufWriter bufWriter
}

// Read forwards reads to the underlying Reader.
func (c *rwConn) Read(p []byte) (int, error) {
	return c.Reader.Read(p)
}

// Write forwards writes to the underlying bufWriter and immediately flushes.
func (c *rwConn) Write(p []byte) (int, error) {
	n, err := c.BufWriter.Write(p)
	if err != nil {
		return n, err
	}
	if err = c.BufWriter.Flush(); err != nil {
		return n, err
	}
	return n, err
}
