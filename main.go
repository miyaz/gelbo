package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var listenPort int
var syncerPort int
var cw ConnectionWatcher

func main() {
	if runtime.GOOS == "linux" {
		go cpuControl(store.resource.CPU.TargetChan)
		go memoryControl(store.resource.Memory.TargetChan)
	}

	rand.Seed(time.Now().UnixNano())

	store.host.IP = getIPAddress()
	store.host.Name, _ = os.Hostname()
	if runOnAws {
		if ip := getEC2MetaData("local-ipv4"); ip != "" {
			store.host.IP = ip
		}
		if name := getEC2MetaData("local-hostname"); name != "" {
			store.host.Name = getEC2MetaData("local-hostname")
		}
	}
	syncer.Nodes[store.host.IP] = store.node

	if syncerMode {
		initSyncer()
		go loopSyncer()
	}

	http.HandleFunc("/exec/", execHandler)
	http.HandleFunc("/syncer/", syncerHandler)
	http.HandleFunc("/monitor/", monitorHandler)
	http.HandleFunc("/", defaultHandler)
	srv := &http.Server{
		Addr:        ":" + strconv.Itoa(listenPort),
		IdleTimeout: 65 * time.Second,
		ConnState:   cw.OnStateChange,
	}
	log.Fatalln(srv.ListenAndServe())
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
	case http.StateActive:
		if _, ok := conns.get(conn.RemoteAddr().String()); !ok {
			conns.set(conn.RemoteAddr().String(), conn)
			atomic.AddInt64(&cw.active, 1)
		}
	case http.StateIdle:
		if _, ok := conns.get(conn.RemoteAddr().String()); ok {
			conns.del(conn.RemoteAddr().String())
			atomic.AddInt64(&cw.active, -1)
		}
	case http.StateHijacked, http.StateClosed:
		if _, ok := conns.get(conn.RemoteAddr().String()); ok {
			conns.del(conn.RemoteAddr().String())
			atomic.AddInt64(&cw.active, -1)
		}
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
	qsMap := r.URL.Query()
	for key, values := range qsMap {
		if key != "cmd" {
			continue
		}
		for _, value := range values {
			// security improvement
			if strings.Index(value, "security-credentials") != -1 {
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
				fmt.Fprintf(w, "%v\n", err)
			}
			fmt.Fprintf(w, "%s\n", string(out))
		}
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := RequestInfo{
		Path:   r.URL.EscapedPath(),
		Query:  r.URL.Query().Encode(),
		Header: combineValues(r.Header),
	}
	reqInfo.Header["Host"] = r.Host
	reqInfo.setIPAddresse(r)
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

	reqSize, _ := io.Copy(ioutil.Discard, r.Body)
	respSize := execAction(w, &respInfo)
	store.node.reflectRequest(reqSize, respSize)
	fmt.Printf("total: %d, active: %d\n", cw.getTotalConns(), cw.getActiveConns())
}

func execAction(w http.ResponseWriter, respInfo *ResponseInfo) int64 {
	respJSON, _ := json.MarshalIndent(*respInfo, "", "  ")
	respSize := len(respJSON)
	if respInfo.Direction.Input.needsAction() {
		if arrayContains(respInfo.Direction.Input.actions, "sleep") {
			sleep, _ := strconv.Atoi(respInfo.Direction.Result.getValue("sleep"))
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
		if arrayContains(respInfo.Direction.Input.actions, "status") {
			status, _ := strconv.Atoi(respInfo.Direction.Result.getValue("status"))
			w.WriteHeader(status)
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
		}
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(respSize))
	if err := writeResponse(w, respSize, respJSON); err != nil {
		fmt.Println(err)
	}
	return int64(respSize)
}

func arrayContains(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
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
				fmt.Println("Current IP address : ", ipnet.IP.String())
				currentIP = ipnet.IP.String()
			}
		}
	}
	return currentIP
}
