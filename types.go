package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var store = NewDataStore()

const orSeparator = " or "

// NewDataStore ... create datastore instance
func NewDataStore() *DataStore {
	_store := &DataStore{
		host:      &HostInfo{},
		node:      NewNodeInfo(),
		resource:  NewResourceInfo(),
		validator: newValidator(),
	}
	_store.RWMutex = &sync.RWMutex{}
	return _store
}

// DataStore ... Variables that use mutex
type DataStore struct {
	*sync.RWMutex
	host      *HostInfo
	node      *NodeInfo
	resource  *ResourceInfo
	validator map[string]*regexp.Regexp
}

func (ds *DataStore) getHostInfo() *HostInfo {
	ds.RLock()
	defer ds.RUnlock()
	host := *ds.host
	return &host
}

// HostInfo ... information of host
type HostInfo struct {
	Name         string `json:"name"`
	IP           string `json:"ip"`
	AZ           string `json:"az,omitempty"`
	InstanceType string `json:"type,omitempty"`
}

// ResourceInfo ... information of os resource
type ResourceInfo struct {
	CPU    ResourceUsage `json:"cpu"`
	Memory ResourceUsage `json:"memory"`
}

// NewResourceInfo ... create resource info instance
func NewResourceInfo() *ResourceInfo {
	resrc := &ResourceInfo{
		CPU:    ResourceUsage{&sync.RWMutex{}, make(chan float64), 0, 0},
		Memory: ResourceUsage{&sync.RWMutex{}, make(chan float64), 0, 0},
	}
	return resrc
}

// ResourceUsage ... information of os resource usage
type ResourceUsage struct {
	*sync.RWMutex
	TargetChan chan float64 `json:"-"`
	Target     float64      `json:"target"`
	Current    float64      `json:"current"`
}

func (ru *ResourceUsage) getTarget() float64 {
	ru.RLock()
	defer ru.RUnlock()
	return ru.Target
}
func (ru *ResourceUsage) setTarget(value float64) {
	if ru.getTarget() != value {
		ru.Lock()
		defer ru.Unlock()
		if runtime.GOOS == "linux" {
			ru.TargetChan <- value
		}
		ru.Target = value
	}
}
func (ru *ResourceUsage) getCurrent() float64 {
	ru.RLock()
	defer ru.RUnlock()
	return ru.Current
}
func (ru *ResourceUsage) setCurrent(value float64) {
	if ru.getCurrent() != value {
		ru.Lock()
		defer ru.Unlock()
		ru.Current = value
	}
}

// RequestInfo ... information of request
type RequestInfo struct {
	Proto     string            `json:"protocol"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Query     string            `json:"querystring,omitempty"`
	Header    map[string]string `json:"header"`
	ClientIP  string            `json:"clientip"`
	Proxy1IP  string            `json:"proxy1ip,omitempty"`
	Proxy2IP  string            `json:"proxy2ip,omitempty"`
	Proxy3IP  string            `json:"proxy3ip,omitempty"`
	LastHopIP string            `json:"lasthopip,omitempty"`
	TargetIP  string            `json:"targetip"`
	MtlsCert  string            `json:"mtlscert,omitempty"`
}

// Direction ... information of directions
type Direction struct {
	Input  *Commands `json:"input"`
	Result *Commands `json:"result"`
}

// ResponseInfo ... information of response
type ResponseInfo struct {
	Host      HostInfo     `json:"host"`
	Resource  ResourceInfo `json:"resource"`
	Request   RequestInfo  `json:"request"`
	Direction Direction    `json:"direction"`
}

// Commands ... Commands Values
type Commands struct {
	CPU         string `json:"cpu,omitempty"`
	Memory      string `json:"memory,omitempty"`
	Sleep       string `json:"sleep,omitempty"`
	Size        string `json:"size,omitempty"`
	Status      string `json:"status,omitempty"`
	AddHeader   string `json:"addheader,omitempty"`
	DelHeader   string `json:"delheader,omitempty"`
	Chunk       string `json:"chunk,omitempty"`
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	actions     []string
	ifMatches   []string
	ifUnmatches []string
	invalids    []string
	IfClientIP  string `json:"ifclientip,omitempty"`
	IfProxy1IP  string `json:"ifproxy1ip,omitempty"`
	IfProxy2IP  string `json:"ifproxy2ip,omitempty"`
	IfProxy3IP  string `json:"ifproxy3ip,omitempty"`
	IfLasthopIP string `json:"iflasthopip,omitempty"`
	IfTargetIP  string `json:"iftargetip,omitempty"`
	IfHostIP    string `json:"ifhostip,omitempty"`
	IfHost      string `json:"ifhost,omitempty"`
	IfAZ        string `json:"ifaz,omitempty"`
	IfType      string `json:"iftype,omitempty"`
}

func (cmds *Commands) getValue(key string) (ret string) {
	switch key {
	case "cpu":
		ret = cmds.CPU
	case "memory":
		ret = cmds.Memory
	case "sleep":
		ret = cmds.Sleep
	case "size":
		ret = cmds.Size
	case "status":
		ret = cmds.Status
	case "addheader":
		ret = cmds.AddHeader
	case "delheader":
		ret = cmds.DelHeader
	case "chunk":
		ret = cmds.Chunk
	case "stdout":
		ret = cmds.Stdout
	case "stderr":
		ret = cmds.Stderr
	}
	return
}

func (cmds *Commands) setValue(key, value string) {
	switch key {
	case "cpu":
		cmds.CPU = value
	case "memory":
		cmds.Memory = value
	case "sleep":
		cmds.Sleep = value
	case "size":
		cmds.Size = value
	case "status":
		cmds.Status = value
	case "addheader":
		cmds.AddHeader = value
	case "delheader":
		cmds.DelHeader = value
	case "chunk":
		cmds.Chunk = value
	case "stdout":
		cmds.Stdout = value
	case "stderr":
		cmds.Stderr = value
	case "ifclientip":
		cmds.IfClientIP = value
	case "ifproxy1ip":
		cmds.IfProxy1IP = value
	case "ifproxy2ip":
		cmds.IfProxy2IP = value
	case "ifproxy3ip":
		cmds.IfProxy3IP = value
	case "iflasthopip":
		cmds.IfLasthopIP = value
	case "iftargetip":
		cmds.IfTargetIP = value
	case "ifhostip":
		cmds.IfHostIP = value
	case "ifhost":
		cmds.IfHost = value
	case "ifaz":
		cmds.IfAZ = value
	case "iftype":
		cmds.IfType = value
	}
}

func newValidator() map[string]*regexp.Regexp {
	const (
		regexpPercent      = "^(100|[0-9]{1,2})$"
		regexpNumRange     = "^([0-9]+)(?:-([0-9]+))?$"
		regexpStatus       = "^([1-9][0-9]{2})$"
		regexpHeader       = "^([a-zA-Z0-9-]+): .+$"
		regexpHeaderName   = "^([a-zA-Z0-9-]+)$"
		regexpModeOn       = "^(on|1|t|true)$"
		regexpHostname     = "([a-zA-Z0-9-.]+)"
		regexpAZone        = "([a-z]{2}-[a-z]+-[1-9][a-d])"
		regexpInstanceType = "(([a-z0-9]+)\\.([a-z0-9]+))"
		regexpIPv4         = "((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)"
		regexpIPv6         = "(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))"
		regexpAll          = "^(.*)$"
	)
	validator := map[string]*regexp.Regexp{}
	validator["cpu"] = regexp.MustCompile(regexpPercent)
	validator["memory"] = regexp.MustCompile(regexpPercent)
	validator["sleep"] = regexp.MustCompile(regexpNumRange)
	validator["size"] = regexp.MustCompile(regexpNumRange)
	validator["status"] = regexp.MustCompile(regexpStatus)
	validator["addheader"] = regexp.MustCompile(regexpHeader)
	validator["delheader"] = regexp.MustCompile(regexpHeaderName)
	validator["chunk"] = regexp.MustCompile(regexpAll)
	validator["stdout"] = regexp.MustCompile(regexpAll)
	validator["stderr"] = regexp.MustCompile(regexpAll)
	validator["ifhost"] = regexp.MustCompile("^(" + regexpHostname + "(" + orSeparator + regexpHostname + ")*)$")
	validator["ifaz"] = regexp.MustCompile("^(" + regexpAZone + "(" + orSeparator + regexpAZone + ")*)$")
	validator["iftype"] = regexp.MustCompile("^(" + regexpInstanceType + "(" + orSeparator + regexpInstanceType + ")*)$")
	regexpIPv4v6 := fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6)
	validator["ifhostip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	validator["iftargetip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	validator["ifproxy1ip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	validator["ifproxy2ip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	validator["ifproxy3ip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	validator["iflasthopip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	validator["ifclientip"] = regexp.MustCompile("^(" + regexpIPv4v6 + "(" + orSeparator + regexpIPv4v6 + ")*)$")
	return validator
}

func (reqInfo *RequestInfo) validateCommands(mapCmds map[string][]string) *Commands {
	cmds := &Commands{}
	for key, value := range combineValuesWithOr(mapCmds) {
		if re, ok := store.validator[key]; ok {
			cmds.setValue(key, value)
			if len(re.FindStringSubmatch(value)) > 0 {
				if strings.HasPrefix(key, "if") {
					if judgeActualValue(reqInfo.getActualValue(key), value) {
						cmds.ifMatches = append(cmds.ifMatches, key)
					} else {
						cmds.ifUnmatches = append(cmds.ifUnmatches, key)
					}
				} else {
					cmds.actions = append(cmds.actions, key)
				}
			} else {
				cmds.invalids = append(cmds.invalids, key)
			}
		}
	}
	return cmds
}

func judgeActualValue(actualValue, value string) bool {
	if strings.Contains(value, orSeparator) {
		for _, v := range strings.Split(value, orSeparator) {
			if actualValue == v {
				return true
			}
		}
		return false
	}
	return actualValue == value
}

func combineValuesWithOr(input map[string][]string) map[string]string {
	output := map[string]string{}
	for key := range input {
		output[key] = strings.Join(input[key], orSeparator)
	}
	return output
}

func (reqInfo *RequestInfo) setIPAddress(r *http.Request) {
	cs, ok := csMaps.get(r.RemoteAddr)
	if ok {
		reqInfo.TargetIP = extractIPAddress(cs.conn.LocalAddr().String())
	} else {
		reqInfo.TargetIP = extractIPAddress(store.host.IP)
	}
	xff := splitXFF(r.Header.Get("X-Forwarded-For"))
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
		reqInfo.ClientIP = extractIPAddress(r.RemoteAddr)
	} else {
		reqInfo.ClientIP = extractIPAddress(xff[0])
		reqInfo.LastHopIP = extractIPAddress(r.RemoteAddr)
		// use elb
		store.node.Lock()
		defer store.node.Unlock()
		store.node.ELBs[reqInfo.LastHopIP] = remoteNodes.m[reqInfo.LastHopIP]
	}
}

func extractIPAddress(ipport string) string {
	var ipaddr string
	if len(strings.Split(ipport, ":")) > 2 { // ipv6
		portExists := 0
		if strings.HasPrefix(ipport, "[") && !strings.HasSuffix(ipport, "]") {
			portExists = 1
		}
		ipaddr = strings.Join(strings.Split(ipport, ":")[:len(strings.Split(ipport, ":"))-portExists], ":")
		ipaddr = strings.Trim(ipaddr, "[]")
	} else { // ipv4
		if strings.Index(ipport, ":") != -1 {
			ipaddr = strings.Split(ipport, ":")[0]
		} else {
			ipaddr = ipport
		}
	}
	return ipaddr
}

func extractPort(ipport string) int {
	var port string
	if len(strings.Split(ipport, ":")) > 2 { // ipv6
		if strings.HasPrefix(ipport, "[") && !strings.HasSuffix(ipport, "]") {
			port = strings.Join(strings.Split(ipport, ":")[len(strings.Split(ipport, ":"))-1:], ":")
		}
	} else { // ipv4
		if strings.Index(ipport, ":") != -1 {
			port = strings.Split(ipport, ":")[1]
		}
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		portNum = 0
	}
	return portNum
}

func splitXFF(xffStr string) []string {
	if xffStr == "" {
		return []string{}
	}
	xff := strings.Split(xffStr, ",")
	for i := range xff {
		xff[i] = strings.TrimSpace(xff[i])
	}
	return xff
}

func (cmds *Commands) evaluate() *Commands {
	resultCmds := Commands{}
	for _, invalid := range cmds.invalids {
		resultCmds.setValue(invalid, "invalid")
	}
	if len(cmds.actions) == 0 {
		return &resultCmds
	}
	for _, ifMatch := range cmds.ifMatches {
		resultCmds.setValue(ifMatch, "matched")
	}
	for _, ifUnmatch := range cmds.ifUnmatches {
		resultCmds.setValue(ifUnmatch, "unmatched")
	}
	// action evaluation
	for _, action := range cmds.actions {
		resultCmds.setValue(action, cmds.getActionValue(action))
	}
	return &resultCmds
}

func (cmds *Commands) needsAction() bool {
	if len(cmds.actions) == 0 {
		return false
	}
	if len(cmds.ifUnmatches) != 0 {
		return false
	}
	if len(cmds.invalids) != 0 {
		return false
	}
	return true
}

func (cmds *Commands) getActionValue(key string) (ret string) {
	if key == "sleep" || key == "size" || key == "repeat" {
		values := strings.Split(cmds.getValue(key), "-")
		if len(values) == 1 {
			ret = cmds.getValue(key)
		} else {
			minValue, _ := strconv.Atoi(values[0])
			maxValue, _ := strconv.Atoi(values[1])
			if minValue > maxValue {
				tmp := minValue
				minValue = maxValue
				maxValue = tmp
			}
			ret = strconv.Itoa(minValue + rand.Intn(maxValue-minValue+1))
		}
	} else if key == "chunk" {
		ret = "chunked when using HTTP/1.1"
	} else {
		ret = cmds.getValue(key)
	}
	return
}

func (reqInfo *RequestInfo) getActualValue(key string) (ret string) {
	switch key {
	case "ifclientip":
		ret = reqInfo.ClientIP
	case "ifproxy1ip":
		ret = reqInfo.Proxy1IP
	case "ifproxy2ip":
		ret = reqInfo.Proxy2IP
	case "ifproxy3ip":
		ret = reqInfo.Proxy3IP
	case "iflasthopip":
		ret = reqInfo.LastHopIP
	case "iftargetip":
		ret = reqInfo.TargetIP
	case "ifhostip":
		ret = store.host.IP
	case "ifhost":
		ret = store.host.Name
	case "ifaz":
		ret = store.host.AZ
	case "iftype":
		ret = store.host.InstanceType
	}
	return
}

var csMaps = NewConnStateMap()

// ConnState ... store connection state and some attributes
type ConnState struct {
	mu        *sync.RWMutex
	conn      net.Conn
	reuse     int64
	prevState http.ConnState
	curState  http.ConnState
}

func (cs *ConnState) updateState(state http.ConnState) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.prevState = cs.curState
	cs.curState = state
}

// ConnStateMap ... map of connections with exclusive control
type ConnStateMap struct {
	*sync.RWMutex
	m map[string]*ConnState
}

// NewConnStateMap ... create ConnStateMap instance
func NewConnStateMap() *ConnStateMap {
	return &ConnStateMap{&sync.RWMutex{}, make(map[string]*ConnState)}
}

func (csm *ConnStateMap) set(k string, v net.Conn) {
	csm.Lock()
	defer csm.Unlock()
	csm.m[k] = &ConnState{
		mu:        &sync.RWMutex{},
		conn:      v,
		reuse:     -1, // set var[reuse] to -1 as initial value because it will be set to 0 the first time it is used
		prevState: http.StateNew,
		curState:  http.StateNew,
	}
}
func (csm *ConnStateMap) get(k string) (*ConnState, bool) {
	csm.RLock()
	defer csm.RUnlock()
	v, ok := csm.m[k]
	return v, ok
}
func (csm *ConnStateMap) del(k string) {
	csm.Lock()
	defer csm.Unlock()
	delete(csm.m, k)
}

var remoteNodes = NewRemoteNodeMap()

// RemoteNodeMap ... map of remote nodes with exclusive control
type RemoteNodeMap struct {
	*sync.RWMutex
	m map[string]*NodeInfo
}

// NewRemoteNodeMap ... create RemoteNodeMap instance
func NewRemoteNodeMap() *RemoteNodeMap {
	return &RemoteNodeMap{&sync.RWMutex{}, make(map[string]*NodeInfo)}
}

func (rnm *RemoteNodeMap) addTotalConns(remoteAddr string, cnt int64) {
	rnm.Lock()
	defer rnm.Unlock()
	now := time.Now().UnixNano()
	if _, ok := rnm.m[remoteAddr]; !ok {
		rnm.m[remoteAddr] = NewNodeInfo()
		rnm.m[remoteAddr].CreatedAt = now
	}
	rnm.m[remoteAddr].addTotalConns(cnt)
}
func (rnm *RemoteNodeMap) addActiveConns(remoteAddr string, cnt int64) {
	rnm.Lock()
	defer rnm.Unlock()
	rnm.m[remoteAddr].addActiveConns(cnt)
}

var headerMap = NewHeaderMap()

// HeaderMap ... map of response header with exclusive control
type HeaderMap struct {
	*sync.RWMutex
	m map[string]string
}

// NewHeaderMap ... create HeaderMap instance
func NewHeaderMap() *HeaderMap {
	return &HeaderMap{&sync.RWMutex{}, make(map[string]string)}
}

func (hm *HeaderMap) add(key, value string) {
	hm.Lock()
	defer hm.Unlock()
	hm.m[strings.ToLower(key)] = value
}
func (hm *HeaderMap) getAll() map[string]string {
	m := make(map[string]string)
	hm.RLock()
	defer hm.RUnlock()
	for key, value := range hm.m {
		m[key] = value
	}
	return m
}
func (hm *HeaderMap) del(key string) {
	hm.Lock()
	defer hm.Unlock()
	lkey := strings.ToLower(key)
	if _, ok := hm.m[lkey]; ok {
		delete(hm.m, lkey)
	}
}
