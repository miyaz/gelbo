package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	_ "embed"

	proxyproto "github.com/pires/go-proxyproto"
	"github.com/rs/zerolog"
	"github.com/smallstep/certinfo"
	"golang.org/x/net/http2"
)

var (
	httpPort    int
	httpsPort   int
	noLogFlag   bool
	execFlag    bool
	proxyFlag   bool
	idleTimeout int
	cw          ConnectionWatcher

	//go:embed cert/server-cert.pem
	certData []byte
	//go:embed cert/server-key.pem
	keyData []byte
)

// PPWrapListenAndServeProps ... ListenAndServeProps for Proxy Protocol
type PPWrapListenAndServeProps struct {
	Srv    *http.Server
	UseTLS bool
}

// PPWrapListenAndServe ... ListenAndServeWrapper for Proxy Protocol
func PPWrapListenAndServe(props *PPWrapListenAndServeProps) error {
	ln, err := net.Listen("tcp", props.Srv.Addr)
	if err != nil {
		panic(err)
	}

	proxyListener := &proxyproto.Listener{
		Listener:          ln,
		ReadHeaderTimeout: -1,
	}
	defer proxyListener.Close()

	if props.UseTLS == true {
		return props.Srv.ServeTLS(proxyListener, "", "")
	}
	return props.Srv.Serve(proxyListener)
}

func main() {
	if noLogFlag {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldInteger = true

	if runtime.GOOS == "linux" {
		go cpuControl(store.resource.CPU.TargetChan)
		go memoryControl(store.resource.Memory.TargetChan)
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))

	store.host.IP = getIPAddress()
	store.host.Name, _ = os.Hostname()
	if runOnEC2 {
		if ip := getEC2MetaData("local-ipv4"); ip != "" {
			store.host.IP = ip
		}
		if name := getEC2MetaData("local-hostname"); name != "" {
			store.host.Name = getEC2MetaData("local-hostname")
		}
	}

	go hub.run()
	router := http.NewServeMux()
	if execFlag {
		router.HandleFunc("/exec/", handlerWrapper(execHandler))
	}
	router.HandleFunc("/stop/", stopHandler)
	router.HandleFunc("/env/", handlerWrapper(envHandler))
	router.HandleFunc("/chat/", handlerWrapper(chatPageHandler))
	router.HandleFunc("/ws/", wsHandler)
	router.HandleFunc("/monitor/", monitorHandler)
	router.HandleFunc("/", handlerWrapper(defaultHandler))
	h2cWrapper := &HandlerH2C{
		Handler:  router,
		H2Server: &http2.Server{},
	}

	tlssrv := &http.Server{
		Addr:        ":" + strconv.Itoa(httpsPort),
		IdleTimeout: time.Duration(idleTimeout) * time.Second,
		ConnState:   cw.OnStateChange,
		Handler:     h2cWrapper,
		TLSConfig:   loadTLSConfig(),
		ErrorLog:    log.New(io.Discard, "", 0),
	}
	go func() {
		var err error
		if proxyFlag {
			err = PPWrapListenAndServe(&PPWrapListenAndServeProps{
				Srv:    tlssrv,
				UseTLS: true,
			})
		} else {
			err = tlssrv.ListenAndServeTLS("", "")
		}
		log.Fatalln(err)
	}()

	httpSrv := &http.Server{
		Addr:        ":" + strconv.Itoa(httpPort),
		IdleTimeout: time.Duration(idleTimeout) * time.Second,
		ConnState:   cw.OnStateChange,
		Handler:     h2cWrapper,
		ErrorLog:    log.New(io.Discard, "", 0),
	}
	var err error
	if proxyFlag {
		err = PPWrapListenAndServe(&PPWrapListenAndServeProps{
			Srv:    httpSrv,
			UseTLS: false,
		})
	} else {
		err = httpSrv.ListenAndServe()
	}
	log.Fatalln(err)
}

// ConnectionWatcher ... connection counter
type ConnectionWatcher struct {
	total  int64
	active int64
}

// OnStateChange ... records open connections in response to connection
func (cw *ConnectionWatcher) OnStateChange(conn net.Conn, state http.ConnState) {
	remoteAddr := conn.RemoteAddr().String()
	if state == http.StateNew {
		if _, ok := csMaps.get(remoteAddr); ok {
			csMaps.del(remoteAddr)
		}
		csMaps.set(remoteAddr, conn)
		atomic.AddInt64(&cw.total, 1)
		remoteNodes.addTotalConns(extractIPAddress(remoteAddr), 1)
		return
	}

	cs, ok := csMaps.get(remoteAddr)
	if !ok {
		return
	}
	cs.updateState(state)
	switch state {
	case http.StateActive:
		atomic.AddInt64(&cw.active, 1)
		remoteNodes.addActiveConns(extractIPAddress(remoteAddr), 1)
	case http.StateIdle:
		atomic.AddInt64(&cw.active, -1)
		remoteNodes.addActiveConns(extractIPAddress(remoteAddr), -1)
	case http.StateHijacked:
		atomic.AddInt64(&cw.active, -1)
		remoteNodes.addActiveConns(extractIPAddress(remoteAddr), -1)
		atomic.AddInt64(&cw.total, -1)
		remoteNodes.addTotalConns(extractIPAddress(remoteAddr), -1)
	case http.StateClosed:
		if cs.prevState == http.StateActive {
			atomic.AddInt64(&cw.active, -1)
			remoteNodes.addActiveConns(extractIPAddress(remoteAddr), -1)
		}
		remoteNodes.addTotalConns(extractIPAddress(remoteAddr), -1)
		atomic.AddInt64(&cw.total, -1)
	}
}

func (cw *ConnectionWatcher) getTotalConns() int64 {
	return atomic.LoadInt64(&cw.total)
}

func (cw *ConnectionWatcher) getActiveConns() int64 {
	return atomic.LoadInt64(&cw.active)
}

func handlerWrapper(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reuse int64
		if cs, ok := csMaps.get(r.RemoteAddr); ok {
			reuse = atomic.AddInt64(&cs.reuse, 1)
		}
		httpLogger, _ := r.Context().Value("logger").(*HttpLogger)
		httpLogger.init(r, reuse)
		fn(w, r)
		httpLogger.log()
	}
}

func execHandler(w http.ResponseWriter, r *http.Request) {
	qsMap := r.URL.Query()
	var respStrs []string
	for key, values := range qsMap {
		if key != "cmd" {
			continue
		}
		for _, value := range values {
			// preventing access to credentials (on ec2/ecs)
			if strings.Index(value, "credentials/") != -1 {
				continue
			}
			args := strings.Split(value, " ")
			var out []byte
			var err error
			if len(args) == 1 {
				out, err = exec.Command(args[0]).Output()
			} else {
				out, err = exec.Command(args[0], args[1:]...).Output()
			}
			if err != nil {
				respStrs = append(respStrs, fmt.Sprintf("%v\n", err))
			}
			respStrs = append(respStrs, fmt.Sprintf("%s\n", string(out)))
		}
	}
	respBody := strings.Join(respStrs, "")
	fmt.Fprintf(w, "%s", respBody)

	setRespSizeForLogger(int64(len(respBody)), r)
	setStatusForLogger(http.StatusOK, r)
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	qsMap := r.URL.Query()
	var respStrs []string
	for key, values := range qsMap {
		if key != "key" {
			continue
		}
		for _, value := range values {
			// preventing access to credentials in environment variables
			if strings.Index(value, "ACCESS_KEY") != -1 {
				continue
			}
			respStrs = append(respStrs, fmt.Sprintf("%s\n", os.Getenv(value)))
		}
	}
	respBody := strings.Join(respStrs, "")
	fmt.Fprintf(w, "%s", respBody)

	setRespSizeForLogger(int64(len(respBody)), r)
	setStatusForLogger(http.StatusOK, r)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	log.Fatalf("stop request received")
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	proto, _ := r.Context().Value("proto").(string)
	queryStr, _ := url.QueryUnescape(r.URL.Query().Encode())
	reqHeaders := combineValues(r.Header)
	reqInfo := RequestInfo{
		Proto:  proto,
		Method: r.Method,
		Path:   r.URL.EscapedPath(),
		Query:  queryStr,
		Header: reqHeaders,
	}
	// add (decoded) mtls cert text info
	if mtlsCert := getMtlsCert(reqHeaders); mtlsCert != "" {
		reqInfo.MtlsCert = decodeMtlsCert(mtlsCert)
	}
	reqInfo.Header["Host"] = r.Host
	reqInfo.setIPAddress(r)
	respInfo := ResponseInfo{
		Host: *store.getHostInfo(),
		Resource: ResourceInfo{
			CPU: ResourceUsage{
				Target:  store.resource.CPU.getTarget(),
				Current: store.resource.CPU.getCurrent(),
			},
			Memory: ResourceUsage{
				Target:  store.resource.Memory.getTarget(),
				Current: store.resource.Memory.getCurrent(),
			},
		},
		Request:   reqInfo,
		Direction: Direction{},
	}

	inputQs := reqInfo.validateQueryString(r.URL.Query())
	resultQs := inputQs.evaluate(&reqInfo)
	respInfo.Direction.Input = inputQs
	respInfo.Direction.Result = resultQs

	reqSize, _ := io.Copy(io.Discard, r.Body)
	respSize, statusCode := execAction(w, r, &respInfo)
	store.node.reflectRequest(reqSize, respSize)
	remoteAddr := extractIPAddress(r.RemoteAddr)
	remoteNodes.m[remoteAddr].reflectRequest(reqSize, respSize)
	setRespSizeForLogger(respSize, r)
	setStatusForLogger(statusCode, r)
}

func combineValues(input map[string][]string) map[string]string {
	output := map[string]string{}
	for key := range input {
		output[key] = strings.Join(input[key], ", ")
	}
	return output
}

func execAction(w http.ResponseWriter, r *http.Request, respInfo *ResponseInfo) (int64, int) {
	respJSON, _ := jsonMarshalIndent(*respInfo)
	respSize := len(respJSON)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(respSize))
	statusCode := http.StatusOK
	chunkFlag := false
	if respInfo.Direction.Input.needsAction() {
		if arrayContains(respInfo.Direction.Input.actions, "sleep") {
			sleep, _ := strconv.Atoi(respInfo.Direction.Result.getValue("sleep"))
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
		if arrayContains(respInfo.Direction.Input.actions, "disconnect") {
			proto, _ := r.Context().Value("proto").(string)
			disconnect(w, r.RemoteAddr, proto, respInfo.Direction.Result.getValue("disconnect") == "rst")
			return 0, 0
		}
		if arrayContains(respInfo.Direction.Input.actions, "status") {
			statusCode, _ = strconv.Atoi(respInfo.Direction.Result.getValue("status"))
		}
		if arrayContains(respInfo.Direction.Input.actions, "cpu") {
			cpu, _ := strconv.ParseFloat(respInfo.Direction.Result.getValue("cpu"), 64)
			store.resource.CPU.setTarget(cpu)
		}
		if arrayContains(respInfo.Direction.Input.actions, "memory") {
			memory, _ := strconv.ParseFloat(respInfo.Direction.Result.getValue("memory"), 64)
			store.resource.Memory.setTarget(memory)
		}
		if arrayContains(respInfo.Direction.Input.actions, "size") {
			size, _ := strconv.Atoi(respInfo.Direction.Result.getValue("size"))
			respSize = size
			w.Header().Set("Content-Length", strconv.Itoa(respSize))
		}
		if arrayContains(respInfo.Direction.Input.actions, "addheader") {
			addHeader := strings.SplitN(respInfo.Direction.Result.getValue("addheader"), ":", 2)
			headerMap.add(addHeader[0], addHeader[1])
		}
		if arrayContains(respInfo.Direction.Input.actions, "delheader") {
			headerMap.del(respInfo.Direction.Result.getValue("delheader"))
		}
		if arrayContains(respInfo.Direction.Input.actions, "chunk") {
			chunkFlag = true
			w.Header().Del("Content-Length")
		}
		if arrayContains(respInfo.Direction.Input.actions, "stdout") {
			fmt.Printf("%s\n", respInfo.Direction.Result.getValue("stdout"))
		}
		if arrayContains(respInfo.Direction.Input.actions, "stderr") {
			fmt.Fprintf(os.Stderr, "%s\n", respInfo.Direction.Result.getValue("stderr"))
		}
	}
	for key, value := range headerMap.getAll() {
		w.Header().Add(key, value)
	}
	w.WriteHeader(statusCode)
	var err error
	if chunkFlag && r.Proto == "HTTP/1.1" {
		err = writeChunkedResponse(w, respSize, respJSON)
	} else {
		err = writeResponse(w, respSize, respJSON)
	}
	if err != nil {
		fmt.Println(err)
	}
	return int64(respSize), statusCode
}

func disconnect(w http.ResponseWriter, remoteAddr string, proto string, force bool) {
	if cs, ok := csMaps.get(remoteAddr); ok {
		closeConnection(cs.conn, force)
		// In the case of h2c, the number of connections is already decreased at the time of hijack
		// so below only decrease it for non-h2c.
		if proto != "h2c" {
			if cs.prevState == http.StateActive {
				atomic.AddInt64(&cw.active, -1)
				remoteNodes.addActiveConns(extractIPAddress(remoteAddr), -1)
			}
			if cs.prevState != http.StateClosed {
				atomic.AddInt64(&cw.total, -1)
				remoteNodes.addTotalConns(extractIPAddress(remoteAddr), -1)
			}
		}
		csMaps.del(remoteAddr)
	}
}

func closeConnection(conn net.Conn, force bool) {
	var linger int32
	if !force {
		linger = 1
	}
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		fmt.Println("Not a TCP connection, trying SyscallConn")
		if tlsConn, ok := conn.(*tls.Conn); ok {
			rawConn, err := tlsConn.NetConn().(*net.TCPConn).SyscallConn()
			if err != nil {
				fmt.Println("Failed to get raw connection:", err)
				return
			}
			rawConn.Control(func(fd uintptr) {
				syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{
					Onoff:  1,
					Linger: linger, // Linger を 0 にすると TCP RST が送信される
				})
			})
			conn.Close()
		}
		return
	}

	//tcpConn.SetLinger(0)
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		fmt.Println("Failed to get raw connection:", err)
		return
	}

	rawConn.Control(func(fd uintptr) {
		syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{
			Onoff:  1,
			Linger: linger, // 0 にすると RST が送られる
		})
	})
	tcpConn.Close()
}

func getClientIPAddress(r *http.Request) (clientIP string) {
	xff := splitXFF(r.Header.Get("X-Forwarded-For"))
	if len(xff) == 0 {
		clientIP = extractIPAddress(r.RemoteAddr)
	} else {
		clientIP = extractIPAddress(xff[0])
	}
	return
}

func loadTLSConfig() *tls.Config {
	serverCert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		log.Fatalln(err)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	return config
}

func arrayContains(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

func jsonMarshalIndent(t interface{}) ([]byte, error) {
	marshalBuffer := &bytes.Buffer{}
	encoder := json.NewEncoder(marshalBuffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(t); err != nil {
		return nil, err
	}
	var indentBuffer bytes.Buffer
	err := json.Indent(&indentBuffer, marshalBuffer.Bytes(), "", "  ")
	return indentBuffer.Bytes(), err
}

func getIPAddress() string {
	var currentIP string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				currentIP = ipnet.IP.String()
			}
		}
	}
	return currentIP
}

func getMtlsCert(headers map[string]string) string {
	for key := range headers {
		if key == "X-Amzn-Mtls-Clientcert" || key == "X-Amzn-Mtls-Clientcert-Leaf" {
			if headers[key] != "" {
				return headers[key]
			}
		}
	}
	return ""
}

func decodeMtlsCert(certStr string) string {
	unescapedCertStr, err := url.PathUnescape(certStr)
	if err != nil {
		return "URL decoding error"
	}
	certBytes := []byte(unescapedCertStr)
	certInfo := ""
	certIndex := 0
	for {
		certIndex++
		certInfo += fmt.Sprintf("== [%d] ============\n", certIndex)

		certBlock, rest := pem.Decode(certBytes)
		if certBlock == nil {
			certInfo += "Certificate decoding error"
			return certInfo
		}
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			certInfo += "Certificate parsing error: " + err.Error()
			return certInfo
		}
		result, err := certinfo.CertificateText(cert)
		if err != nil {
			certInfo += "Certificate info converting error: " + err.Error()
			return certInfo
		}
		certInfo += result + "\n"
		if len(rest) == 0 {
			break
		} else {
			certBytes = rest
		}
	}
	return certInfo
}
