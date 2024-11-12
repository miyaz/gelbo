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
	"time"

	_ "embed"

	"github.com/grantae/certinfo"
	proxyproto "github.com/pires/go-proxyproto"
	"github.com/rs/zerolog"
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
		router.HandleFunc("/exec/", execHandler)
	}
	router.HandleFunc("/stop/", stopHandler)
	router.HandleFunc("/env/", envHandler)
	router.HandleFunc("/chat/", chatPageHandler)
	router.HandleFunc("/ws/", wsHandler)
	router.HandleFunc("/monitor/", monitorHandler)
	router.HandleFunc("/", defaultHandler)
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
	switch state {
	case http.StateNew:
		atomic.AddInt64(&cw.total, 1)
		remoteNodes.addTotalConns(extractIPAddress(conn.RemoteAddr().String()), 1)
	case http.StateActive:
		if _, ok := conns.get(conn.RemoteAddr().String()); !ok {
			conns.set(conn.RemoteAddr().String(), conn)
			atomic.AddInt64(&cw.active, 1)
			remoteNodes.addActiveConns(extractIPAddress(conn.RemoteAddr().String()), 1)
		}
	case http.StateIdle:
		if _, ok := conns.get(conn.RemoteAddr().String()); ok {
			conns.del(conn.RemoteAddr().String())
			atomic.AddInt64(&cw.active, -1)
			remoteNodes.addActiveConns(extractIPAddress(conn.RemoteAddr().String()), -1)
		}
	case http.StateHijacked, http.StateClosed:
		if _, ok := conns.get(conn.RemoteAddr().String()); ok {
			conns.del(conn.RemoteAddr().String())
			atomic.AddInt64(&cw.active, -1)
			remoteNodes.addActiveConns(extractIPAddress(conn.RemoteAddr().String()), -1)
		}
		remoteNodes.addTotalConns(extractIPAddress(conn.RemoteAddr().String()), -1)
		atomic.AddInt64(&cw.total, -1)
	}
}

func (cw *ConnectionWatcher) getTotalConns() int64 {
	return atomic.LoadInt64(&cw.total)
}

func (cw *ConnectionWatcher) getActiveConns() int64 {
	return atomic.LoadInt64(&cw.active)
}

func execHandler(w http.ResponseWriter, r *http.Request) {
	reqtime := time.Now()

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

	httpLog(reqtime, int64(len(respBody)), http.StatusOK, r)
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	reqtime := time.Now()

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

	httpLog(reqtime, int64(len(respBody)), http.StatusOK, r)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	log.Fatalf("stop request received")
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	reqtime := time.Now()

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
	respSize, statusCode := execAction(w, &respInfo)

	store.node.reflectRequest(reqSize, respSize)
	remoteAddr := extractIPAddress(r.RemoteAddr)
	remoteNodes.m[remoteAddr].reflectRequest(reqSize, respSize)

	httpLog(reqtime, respSize, statusCode, r)
}

func combineValues(input map[string][]string) map[string]string {
	output := map[string]string{}
	for key := range input {
		output[key] = strings.Join(input[key], ", ")
	}
	return output
}

func execAction(w http.ResponseWriter, respInfo *ResponseInfo) (int64, int) {
	//respJSON, _ := json.MarshalIndent(*respInfo, "", "  ")
	respJSON, _ := jsonMarshalIndent(*respInfo)
	respSize := len(respJSON)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(respSize))
	statusCode := http.StatusOK
	if respInfo.Direction.Input.needsAction() {
		if arrayContains(respInfo.Direction.Input.actions, "sleep") {
			sleep, _ := strconv.Atoi(respInfo.Direction.Result.getValue("sleep"))
			time.Sleep(time.Duration(sleep) * time.Millisecond)
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
	if err := writeResponse(w, respSize, respJSON); err != nil {
		fmt.Println(err)
	}
	return int64(respSize), statusCode
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

func httpLog(reqtime time.Time, respSize int64, statusCode int, r *http.Request) {
	proto, _ := r.Context().Value("proto").(string)
	logger := zerolog.New(os.Stdout).With().Time("reqtime", reqtime).Str("proto", proto).Logger()

	queryStr, _ := url.QueryUnescape(r.URL.Query().Encode())
	remotePort := extractPort(r.RemoteAddr)
	remoteAddr := extractIPAddress(r.RemoteAddr)
	reqSize, _ := io.Copy(io.Discard, r.Body)

	// request logging
	restime := time.Now()
	logger = logger.With().
		Str("method", r.Method).
		Str("path", r.URL.EscapedPath()).
		Str("qstr", queryStr).
		Str("clientip", getClientIPAddress(r)).
		Str("srcip", remoteAddr).
		Int("srcport", remotePort).
		Int64("reqsize", reqSize).
		Int64("size", respSize).
		Int("status", statusCode).
		Time("time", restime).
		Dur("duration", restime.Sub(reqtime)).
		Logger()
	logger.Log().Msg("")
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
