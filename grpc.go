package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/miyaz/gelbo/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	Unary = iota
	ClientStream
	ServerStream
	BidiStream
)

var (
	headerMDMap  = NewHeaderMap()
	trailerMDMap = NewHeaderMap()
	grpcInterval int
)

type grpcServer struct {
	pb.UnimplementedGelboServiceServer
}

func NewGrpcServer() *grpcServer {
	return &grpcServer{}
}

func startGrpcServer() {
	kaep := keepalive.EnforcementPolicy{
		PermitWithoutStream: true,
	}
	kasp := keepalive.ServerParameters{
		Time: time.Duration(grpcInterval) * time.Second,
	}
	grpcSrv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	)
	grpcsSrv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.Creds(credentials.NewTLS(loadTLSConfig())),
	)

	pb.RegisterGelboServiceServer(grpcSrv, NewGrpcServer())
	pb.RegisterGelboServiceServer(grpcsSrv, NewGrpcServer())

	// enable server reflection
	reflection.Register(grpcSrv)
	reflection.Register(grpcsSrv)

	lnCnf := net.ListenConfig{
		KeepAlive: time.Duration(probeInterval) * time.Second,
	}
	if probeInterval == 0 {
		lnCnf.KeepAlive = -1
	}
	go func() {
		if grpcLn, err := lnCnf.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", grpcPort)); err != nil {
			log.Fatalln(err)
		} else {
			err = grpcSrv.Serve(grpcLn)
			log.Fatalln(err)
		}
	}()
	if grpcsLn, err := lnCnf.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", grpcsPort)); err != nil {
		log.Fatalln(err)
	} else {
		err = grpcsSrv.Serve(grpcsLn)
		log.Fatalln(err)
	}
}

func (s *grpcServer) Unary(ctx context.Context, req *pb.GelboRequest) (*pb.GelboResponse, error) {
	log.SetFlags(log.Lmicroseconds)
	sendChan := make(chan *pb.GelboResponse)
	errChan := make(chan error, 1)
	wg := NewWaitGroup()
	wg.Add(1)

	go s.handler(Unary, ctx, req, sendChan, errChan, wg)

	for {
		select {
		case resp := <-sendChan:
			return resp, nil
		case err := <-errChan:
			return nil, err
		}
	}
}

func (s *grpcServer) ClientStream(stream pb.GelboService_ClientStreamServer) error {
	log.SetFlags(log.Lmicroseconds)
	sendChan := make(chan *pb.GelboResponse)
	errChan := make(chan error, 1)
	wg := NewWaitGroup()
	var latestReq *pb.GelboRequest

	for {
		wg.Add(1)
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			go s.handler(ClientStream, stream.Context(), latestReq, sendChan, errChan, wg)
			break
		} else {
			latestReq = req
		}
	}
	for {
		select {
		case resp, ok := <-sendChan:
			if !ok {
				return nil
			}
			wg.Done()
			err := stream.SendAndClose(resp)
			return err
		case err := <-errChan:
			return err
		}
	}
}

func (s *grpcServer) ServerStream(req *pb.GelboRequest, stream pb.GelboService_ServerStreamServer) error {
	log.SetFlags(log.Lmicroseconds)
	sendChan := make(chan *pb.GelboResponse)
	errChan := make(chan error, 1)
	wg := NewWaitGroup()
	wg.Add(1)

	go s.handler(ServerStream, stream.Context(), req, sendChan, errChan, wg)
	go s.sender(stream, sendChan, errChan, wg)

	wg.Wait()
	close(sendChan)
	for {
		select {
		case err := <-errChan:
			return err
		}
	}
}

func (s *grpcServer) BidiStream(stream pb.GelboService_BidiStreamServer) error {
	log.SetFlags(log.Lmicroseconds)
	recvChan := make(chan *pb.GelboRequest)
	sendChan := make(chan *pb.GelboResponse)
	errChan := make(chan error)
	wg := NewWaitGroup()

	go s.receiver(stream, recvChan, errChan)
	go s.sender(stream, sendChan, errChan, wg)

	for {
		select {
		case req, ok := <-recvChan:
			if !ok {
				wg.Wait()
				select {
				case err := <-errChan:
					return err
				default:
					close(sendChan)
					return nil
				}
			}
			wg.Add(1)
			go s.handler(BidiStream, stream.Context(), req, sendChan, errChan, wg)
		case err := <-errChan:
			return err
		}
	}
}

func (s *grpcServer) handler(mode int, ctx context.Context, req *pb.GelboRequest, sendChan chan *pb.GelboResponse, errChan chan error, wg *WaitGroup) {
	reqInfo := NewRequestInfoFromContext(ctx, req)
	inputCmds := reqInfo.validateCommandsForGrpc(mode, req)
	resultCmds := inputCmds.evaluate()

	if inputCmds.needsAction() {
		if arrayContains(inputCmds.actions, "noop") {
			if mode == Unary || mode == ClientStream {
				errChan <- nil
			}
			wg.Done()
			return
		}
		if inputCmds.Repeat != "" {
			repeat, _ := strconv.Atoi(resultCmds.getValue("repeat"))
			for i := 1; i < repeat; i++ {
				resultCmds.Repeat = strconv.Itoa(repeat)
				if err := execGrpcAction(reqInfo, inputCmds, resultCmds); err != nil {
					errChan <- err
					wg.Done()
					return
				}

				wg.Add(1)
				sendChan <- createResponse(reqInfo, inputCmds, resultCmds)
				resultCmds = inputCmds.evaluate()
			}
		}
		if err := execGrpcAction(reqInfo, inputCmds, resultCmds); err != nil {
			wg.Done()
			errChan <- err
			return
		}
	}
	grpc.SetHeader(ctx, metadata.New(headerMDMap.getAll()))
	grpc.SetTrailer(ctx, metadata.New(trailerMDMap.getAll()))
	sendChan <- createResponse(reqInfo, inputCmds, resultCmds)
}

func (reqInfo *RequestInfo) validateCommandsForGrpc(mode int, req *pb.GelboRequest) *Commands {
	inputCmds := reqInfo.validateCommands(convRequestToMap(req))
	if arrayContains(inputCmds.actions, "repeat") {
		var isRepeatInvalid bool
		if mode == Unary || mode == ClientStream {
			isRepeatInvalid = true
		} else {
			values := strings.Split(inputCmds.getValue("repeat"), "-")
			minValue := 0
			if len(values) == 1 {
				minValue, _ = strconv.Atoi(values[0])
			} else {
				minValue, _ = strconv.Atoi(values[0])
				maxValue, _ := strconv.Atoi(values[1])
				if minValue > maxValue {
					tmp := minValue
					minValue = maxValue
					maxValue = tmp
				}
				if minValue == 0 {
					isRepeatInvalid = true
				}
			}
		}
		if isRepeatInvalid {
			inputCmds.invalids = append(inputCmds.invalids, "repeat")
			newActions := []string{}
			for _, act := range inputCmds.actions {
				if act != "repeat" {
					newActions = append(newActions, act)
				}
			}
			inputCmds.actions = newActions
		}
	}
	return inputCmds
}

func execGrpcAction(reqInfo *RequestInfo, inputCmds, resultCmds *Commands) error {
	if inputCmds.needsAction() {
		if arrayContains(inputCmds.actions, "sleep") {
			sleep, _ := strconv.Atoi(resultCmds.getValue("sleep"))
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
		if arrayContains(inputCmds.actions, "cpu") {
			cpu, _ := strconv.ParseFloat(resultCmds.getValue("cpu"), 64)
			store.resource.CPU.setTarget(cpu)
		}
		if arrayContains(inputCmds.actions, "memory") {
			memory, _ := strconv.ParseFloat(resultCmds.getValue("memory"), 64)
			store.resource.Memory.setTarget(memory)
		}
		if arrayContains(inputCmds.actions, "addheader") {
			addHeader := strings.SplitN(resultCmds.getValue("addheader"), ":", 2)
			headerMDMap.add(addHeader[0], addHeader[1])
		}
		if arrayContains(inputCmds.actions, "delheader") {
			headerMDMap.del(resultCmds.getValue("delheader"))
		}
		if arrayContains(inputCmds.actions, "addtrailer") {
			addTrailer := strings.SplitN(resultCmds.getValue("addtrailer"), ":", 2)
			trailerMDMap.add(addTrailer[0], addTrailer[1])
		}
		if arrayContains(inputCmds.actions, "deltrailer") {
			trailerMDMap.del(resultCmds.getValue("deltrailer"))
		}
		if arrayContains(inputCmds.actions, "stdout") {
			fmt.Printf("%s\n", resultCmds.getValue("stdout"))
		}
		if arrayContains(inputCmds.actions, "stderr") {
			fmt.Fprintf(os.Stderr, "%s\n", resultCmds.getValue("stderr"))
		}
		if arrayContains(inputCmds.actions, "code") {
			codeNum, _ := strconv.Atoi(resultCmds.getValue("code"))
			code := getCodeClass(int32(codeNum))
			return status.Error(code, code.String()) // return nil if codeNum is 0(OK)
		}
	}
	return nil
}

type IStream interface {
	Send(*pb.GelboResponse) error
	Recv() (*pb.GelboRequest, error)
}

func (s *grpcServer) receiver(stream interface{}, recvChan chan *pb.GelboRequest, errChan chan error) {
	var recvStream IStream
	recvStream = stream.(IStream)
	for {
		msg, err := recvStream.Recv()
		if errors.Is(err, io.EOF) {
			close(recvChan)
			return
		}
		if err != nil {
			errChan <- err
			return
		}
		recvChan <- msg
	}
}

func (s *grpcServer) sender(stream interface{}, sendChan chan *pb.GelboResponse, errChan chan error, wg *WaitGroup) {
	var sendStream IStream
	sendStream = stream.(IStream)
	for msg := range sendChan {
		if err := sendStream.Send(msg); err != nil {
			errChan <- err
			return
		}
		wg.Done()
	}
	errChan <- nil
}

func createResponse(reqInfo *RequestInfo, inputCmds, resultCmds *Commands) *pb.GelboResponse {
	var data string
	if inputCmds.needsAction() {
		if arrayContains(inputCmds.actions, "size") {
			size, _ := strconv.Atoi(resultCmds.getValue("size"))
			data = string(randBytes(randSrc, size))
		}
		if arrayContains(inputCmds.actions, "dataonly") {
			return &pb.GelboResponse{Data: data}
		}
	}

	return &pb.GelboResponse{
		Host: &pb.HostInfo{
			Name: store.host.Name,
			Ip:   store.host.IP,
			Az:   store.host.AZ,
			Type: store.host.InstanceType,
		},
		Resource: &pb.ResourceInfo{
			Cpu: &pb.ResourceUsage{
				Target:  store.resource.CPU.getTarget(),
				Current: store.resource.CPU.getCurrent(),
			},
			Memory: &pb.ResourceUsage{
				Target:  store.resource.Memory.getTarget(),
				Current: store.resource.Memory.getCurrent(),
			},
		},
		Request: &pb.RequestInfo{
			Protocol:  reqInfo.Proto,
			Method:    reqInfo.Method,
			Header:    convMapToStrList(reqInfo.Header),
			Clientip:  reqInfo.ClientIP,
			Proxy1Ip:  reqInfo.Proxy1IP,
			Proxy2Ip:  reqInfo.Proxy2IP,
			Proxy3Ip:  reqInfo.Proxy3IP,
			Lasthopip: reqInfo.LastHopIP,
			Targetip:  reqInfo.TargetIP,
		},
		Direction: &pb.Direction{
			Input:  convMapToStrList(convCommandsToMap(inputCmds)),
			Result: convMapToStrList(convCommandsToMap(resultCmds)),
		},
		Data: data,
	}
}

// === utilities & small parts

type WaitGroup struct {
	mu  *sync.RWMutex
	wg  *sync.WaitGroup
	cnt int
}

func NewWaitGroup() *WaitGroup {
	wg := &WaitGroup{mu: &sync.RWMutex{}, wg: &sync.WaitGroup{}}
	return wg
}
func (wg *WaitGroup) Add(num int) {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	wg.wg.Add(num)
	wg.cnt += num
}
func (wg *WaitGroup) Done() {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	if wg.cnt > 0 {
		wg.wg.Done()
		wg.cnt--
	}
}
func (wg *WaitGroup) Finish() {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	for i := 0; i < wg.cnt; i++ {
		wg.wg.Done()
	}
	wg.cnt = 0
}
func (wg *WaitGroup) Wait() {
	wg.wg.Wait()
}

func getCodeClass(_code int32) (code codes.Code) {
	switch _code {
	case int32(codes.OK):
		code = codes.OK
	case int32(codes.Canceled):
		code = codes.Canceled
	case int32(codes.Unknown):
		code = codes.Unknown
	case int32(codes.InvalidArgument):
		code = codes.InvalidArgument
	case int32(codes.DeadlineExceeded):
		code = codes.DeadlineExceeded
	case int32(codes.NotFound):
		code = codes.NotFound
	case int32(codes.AlreadyExists):
		code = codes.AlreadyExists
	case int32(codes.PermissionDenied):
		code = codes.PermissionDenied
	case int32(codes.ResourceExhausted):
		code = codes.ResourceExhausted
	case int32(codes.FailedPrecondition):
		code = codes.FailedPrecondition
	case int32(codes.Aborted):
		code = codes.Aborted
	case int32(codes.OutOfRange):
		code = codes.OutOfRange
	case int32(codes.Unimplemented):
		code = codes.Unimplemented
	case int32(codes.Internal):
		code = codes.Internal
	case int32(codes.Unavailable):
		code = codes.Unavailable
	case int32(codes.DataLoss):
		code = codes.DataLoss
	case int32(codes.Unauthenticated):
		code = codes.Unauthenticated
	default:
		code = codes.Unknown
	}
	return code
}

func NewRequestInfoFromContext(ctx context.Context, req *pb.GelboRequest) *RequestInfo {
	method, _ := grpc.Method(ctx)
	reqInfo := &RequestInfo{
		Method: method,
	}
	xffStr := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		reqInfo.Header = combineValues(md)
		if xffArray, ok := md["x-forwarded-for"]; ok {
			xffStr = xffArray[0]
		}
	}
	if pr, ok := peer.FromContext(ctx); ok {
		setIPAddress(reqInfo, pr, xffStr)
		reqInfo.Proto = "grpc"
		if extractPort(pr.LocalAddr.String()) == grpcsPort {
			reqInfo.Proto = "grpcs"
		}
	}
	return reqInfo
}

func setIPAddress(reqInfo *RequestInfo, pr *peer.Peer, xffStr string) {
	reqInfo.TargetIP = extractIPAddress(pr.LocalAddr.String())
	xff := splitXFF(xffStr)
	if len(xff) >= 2 {
		reqInfo.Proxy1IP = extractIPAddress(xff[1])
	}
	if len(xff) >= 3 {
		reqInfo.Proxy2IP = extractIPAddress(xff[2])
	}
	if len(xff) >= 4 {
		reqInfo.Proxy3IP = extractIPAddress(xff[3])
	}
	if len(xff) == 0 {
		reqInfo.ClientIP = extractIPAddress(pr.Addr.String())
	} else {
		reqInfo.ClientIP = extractIPAddress(xff[0])
		reqInfo.LastHopIP = extractIPAddress(pr.Addr.String())
	}
}

func convRequestToMap(req *pb.GelboRequest) map[string][]string {
	cmds := map[string]string{
		"cpu":         req.GetCpu(),
		"memory":      req.GetMemory(),
		"sleep":       req.GetSleep(),
		"size":        req.GetSize(),
		"code":        req.GetCode(),
		"addheader":   req.GetAddheader(),
		"delheader":   req.GetDelheader(),
		"addtrailer":  req.GetAddtrailer(),
		"deltrailer":  req.GetDeltrailer(),
		"repeat":      req.GetRepeat(),
		"dataonly":    req.GetDataonly(),
		"noop":        req.GetNoop(),
		"ifclientip":  req.GetIfclientip(),
		"ifproxy1ip":  req.GetIfproxy1Ip(),
		"ifproxy2ip":  req.GetIfproxy2Ip(),
		"ifproxy3ip":  req.GetIfproxy3Ip(),
		"iflasthopip": req.GetIflasthopip(),
		"iftargetip":  req.GetIftargetip(),
		"ifhostip":    req.GetIfhostip(),
		"ifhost":      req.GetIfhost(),
		"ifaz":        req.GetIfaz(),
		"iftype":      req.GetIftype(),
	}
	cmdsMap := map[string][]string{}
	for key, value := range cmds {
		if value != "" {
			cmdsMap[key] = []string{value}
		}
	}
	return cmdsMap
}

func convCommandsToMap(cmds *Commands) map[string]string {
	tmpMap := map[string]string{
		"cpu":         cmds.CPU,
		"memory":      cmds.Memory,
		"sleep":       cmds.Sleep,
		"size":        cmds.Size,
		"code":        cmds.Code,
		"addheader":   cmds.AddHeader,
		"delheader":   cmds.DelHeader,
		"addtrailer":  cmds.AddTrailer,
		"deltrailer":  cmds.DelTrailer,
		"repeat":      cmds.Repeat,
		"dataonly":    cmds.DataOnly,
		"noop":        cmds.Noop,
		"ifclientip":  cmds.IfClientIP,
		"ifproxy1ip":  cmds.IfProxy1IP,
		"ifproxy2ip":  cmds.IfProxy2IP,
		"ifproxy3ip":  cmds.IfProxy3IP,
		"iflasthopip": cmds.IfLasthopIP,
		"iftargetip":  cmds.IfTargetIP,
		"ifhostip":    cmds.IfHostIP,
		"ifhost":      cmds.IfHost,
		"ifaz":        cmds.IfAZ,
		"iftype":      cmds.IfType,
	}

	cmdsMap := map[string]string{}
	for key, value := range tmpMap {
		if value != "" {
			cmdsMap[key] = value
		}
	}
	return cmdsMap
}

func convMapToStrList(kvMap map[string]string) []string {
	kvs := []string{}
	for key, value := range kvMap {
		kvs = append(kvs, fmt.Sprintf("%s: %s", key, value))
	}
	slices.Sort(kvs)
	return kvs
}
