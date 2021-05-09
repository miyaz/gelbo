package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var listenPort int

func main() {
	go cpuControl(store.resource.CPU.TargetChan)
	go memoryControl(store.resource.Memory.TargetChan)

	flag.IntVar(&listenPort, "port", 9000, "listen port")
	flag.Parse()
	fmt.Println("Listen Port : ", listenPort)

	rand.Seed(time.Now().UnixNano())

	store.host.Name, _ = os.Hostname()
	store.host.IP = getIPAddress()

	http.HandleFunc("/syncer/", syncerHandler)
	http.HandleFunc("/monitor/", monitorHandler)
	http.HandleFunc("/", defaultHandler)
	srv := &http.Server{
		Addr:        ":" + strconv.Itoa(listenPort),
		IdleTimeout: 65 * time.Second,
	}
	log.Fatalln(srv.ListenAndServe())
}

func syncerHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func monitorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	//w.WriteHeader(http.StatusNotFound)
	reqInfo := RequestInfo{
		Path:   r.URL.EscapedPath(),
		Query:  r.URL.Query().Encode(),
		Header: combineValues(r.Header),
	}
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
	actionQs := inputQs.evaluate(&reqInfo)
	respInfo.Direction.Input = inputQs
	respInfo.Direction.Action = actionQs
	execAction(w, &respInfo)
}

func execAction(w http.ResponseWriter, respInfo *ResponseInfo) {
	respJSON, _ := json.MarshalIndent(*respInfo, "", "  ")
	respLength := len(respJSON)
	if respInfo.Direction.Input.needsAction() {
		if arrayContains(respInfo.Direction.Input.actions, "sleep") {
			sleep, _ := strconv.Atoi(respInfo.Direction.Action.getValue("sleep"))
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
		if arrayContains(respInfo.Direction.Input.actions, "status") {
			status, _ := strconv.Atoi(respInfo.Direction.Action.getValue("status"))
			w.WriteHeader(status)
		}
		if arrayContains(respInfo.Direction.Input.actions, "cpu") {
			cpu, _ := strconv.ParseFloat(respInfo.Direction.Action.getValue("cpu"), 64)
			store.resource.CPU.setTarget(cpu)
		}
		if arrayContains(respInfo.Direction.Input.actions, "memory") {
			memory, _ := strconv.ParseFloat(respInfo.Direction.Action.getValue("memory"), 64)
			store.resource.Memory.setTarget(memory)
		}
		if arrayContains(respInfo.Direction.Input.actions, "size") {
			size, _ := strconv.Atoi(respInfo.Direction.Action.getValue("size"))
			respLength = size
		}
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(respLength))
	writeResponse(w, respLength, respJSON)
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
