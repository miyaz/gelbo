package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type HttpLogger struct {
	reqtime    time.Time
	proto      string
	method     string
	path       string
	qstr       string
	clientip   string
	remoteaddr string
	srcip      string
	srcport    int
	reqsize    int64
	size       int64
	status     int
	time       time.Time
	duration   time.Duration
	reuse      int64
}

func (l *HttpLogger) init(r *http.Request, reuse int64) {
	l.reqtime = time.Now()
	l.proto, _ = r.Context().Value("proto").(string)
	l.method = r.Method
	l.path = r.URL.EscapedPath()
	l.qstr, _ = url.QueryUnescape(r.URL.Query().Encode())
	l.clientip = getClientIPAddress(r)
	l.remoteaddr = r.RemoteAddr
	l.reqsize, _ = io.Copy(io.Discard, r.Body)
	l.reuse = reuse
}

func setRespSizeForLogger(respSize int64, r *http.Request) {
	if logger, ok := r.Context().Value("logger").(*HttpLogger); ok {
		logger.size = respSize
	}
}

func setStatusForLogger(status int, r *http.Request) {
	if logger, ok := r.Context().Value("logger").(*HttpLogger); ok {
		logger.status = status
	}
}

func (l *HttpLogger) log() {
	restime := time.Now()
	// request logging
	logger := zerolog.New(os.Stdout).With().
		Time("reqtime", l.reqtime).
		Str("proto", l.proto).
		Str("method", l.method).
		Str("path", l.path).
		Str("qstr", l.qstr).
		Str("clientip", l.clientip).
		Str("srcip", extractIPAddress(l.remoteaddr)).
		Int("srcport", extractPort(l.remoteaddr)).
		Int64("reqsize", l.reqsize).
		Int64("size", l.size).
		Int("status", l.status).
		Time("time", restime).
		Dur("duration", restime.Sub(l.reqtime)).
		Int64("reuse", l.reuse).
		Logger()
	logger.Log().Msg("")
}

func wsLogger(r *http.Request) *zerolog.Logger {
	proto, _ := r.Context().Value("proto").(string)
	remotePort := extractPort(r.RemoteAddr)
	remoteAddr := extractIPAddress(r.RemoteAddr)
	logger := zerolog.New(os.Stdout).With().
		Time("conntime", time.Now()).
		Str("proto", proto).
		Str("clientip", getClientIPAddress(r)).
		Str("srcip", remoteAddr).
		Int("srcport", remotePort).
		Logger()
	return &logger
}
