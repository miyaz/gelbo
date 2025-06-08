package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	pb "github.com/miyaz/gelbo/grpc/pb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
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

type GrpcLogger struct {
	opentime  time.Time
	recvtime  time.Time
	sendtime  time.Time
	closetime time.Time
	proto     string
	mode      string
	method    string
	params    string
	clientip  string
	srcip     string
	srcport   int
}

func initLoggerForUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) *GrpcLogger {
	l := &GrpcLogger{}
	mds := getMDSetFromContext(ctx)
	l.recvtime = time.Now()
	l.proto = "grpc"
	if mds.TargetPort == grpcsPort {
		l.proto = "grpcs"
	}
	l.mode = "unary"
	l.method = info.FullMethod
	if params, ok := req.(*pb.GelboRequest); ok {
		l.params = params.String()
	}
	l.clientip = mds.ClientIP
	l.srcip = mds.SrcIP
	l.srcport = mds.SrcPort
	return l
}

func (l *GrpcLogger) forUnary() *zerolog.Logger {
	logger := zerolog.New(os.Stdout).With().
		Time("recvtime", l.recvtime).
		Str("proto", l.proto).
		Str("mode", l.mode).
		Str("method", l.method).
		Str("params", l.params).
		Str("clientip", l.clientip).
		Str("srcip", l.srcip).
		Int("srcport", l.srcport).
		Logger()
	return &logger
}

func initLoggerForStream(ctx context.Context, info *grpc.StreamServerInfo) *GrpcLogger {
	l := &GrpcLogger{}
	mds := getMDSetFromContext(ctx)
	l.opentime = time.Now()
	l.proto = "grpc"
	if mds.TargetPort == grpcsPort {
		l.proto = "grpcs"
	}
	l.mode = "client"
	if info.IsServerStream && info.IsClientStream {
		l.mode = "bidirect"
	} else if info.IsServerStream {
		l.mode = "server"
	}
	l.method = info.FullMethod
	l.clientip = mds.ClientIP
	l.srcip = mds.SrcIP
	l.srcport = mds.SrcPort
	return l
}

func (l *GrpcLogger) forStream() *zerolog.Logger {
	logger := zerolog.New(os.Stdout).With().
		Time("opentime", l.opentime).
		Str("proto", l.proto).
		Str("mode", l.mode).
		Str("method", l.method).
		Str("clientip", l.clientip).
		Str("srcip", l.srcip).
		Int("srcport", l.srcport).
		Logger()
	return &logger
}
