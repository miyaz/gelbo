package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var store = NewDataStore()

// NewDataStore ... create datastore instance
func NewDataStore() *DataStore {
	_store := &DataStore{
		host: &HostInfo{},
		node: &NodeInfo{&sync.RWMutex{}, time.Now().UnixNano(), true, 0, 0, 0, 0},
		resource: &ResourceInfo{
			ResourceUsage{&sync.RWMutex{}, make(chan float64), 0, 0},
			ResourceUsage{&sync.RWMutex{}, make(chan float64), 0, 0},
		},
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
	Name string `json:"name"`
	IP   string `json:"ip"`
	AZ   string `json:"az,omitempty"`
}

// NodeInfo ... information of node
type NodeInfo struct {
	*sync.RWMutex
	Time      int64 `json:"time"`
	Reachable bool  `json:"reachable"`

	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Count  int64   `json:"count"`
	Bytes  int64   `json:"bytes"`
}

func (ni *NodeInfo) getTime() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Time
}
func (ni *NodeInfo) setTime(_time int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.Time = _time
}
func (ni *NodeInfo) isReachable() bool {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Reachable
}
func (ni *NodeInfo) setReachable(r bool) {
	ni.Lock()
	defer ni.Unlock()
	ni.Reachable = r
}
func (ni *NodeInfo) getCount() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Count
}
func (ni *NodeInfo) countUp() {
	ni.Lock()
	defer ni.Unlock()
	ni.Count++
	ni.Time = time.Now().UnixNano()
}
func (ni *NodeInfo) getBytes() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Bytes
}
func (ni *NodeInfo) addBytes(bytes int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.Bytes += bytes
}
func (ni *NodeInfo) getClone() *NodeInfo {
	ni.RLock()
	defer ni.RUnlock()
	node := *ni
	return &node
}

// ResourceInfo ... information of os resource
type ResourceInfo struct {
	CPU    ResourceUsage `json:"cpu"`
	Memory ResourceUsage `json:"memory"`
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
		ru.TargetChan <- value
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
	Path     string            `json:"path"`
	Query    string            `json:"querystring,omitempty"`
	Header   map[string]string `json:"header"`
	ClientIP string            `json:"clientip"`
	Proxy1IP string            `json:"proxy1ip,omitempty"`
	Proxy2IP string            `json:"proxy2ip,omitempty"`
	TargetIP string            `json:"targetip"`
}

// Direction ... information of directions
type Direction struct {
	Input  *QueryString `json:"input"`
	Action *QueryString `json:"action"`
}

// ResponseInfo ... information of response
type ResponseInfo struct {
	Host      HostInfo     `json:"host"`
	Resource  ResourceInfo `json:"resource"`
	Request   RequestInfo  `json:"request"`
	Direction Direction    `json:"direction"`
}

// QueryString ... QueryString Values
type QueryString struct {
	CPU         string `json:"cpu,omitempty"`
	Memory      string `json:"memory,omitempty"`
	Sleep       string `json:"sleep,omitempty"`
	Size        string `json:"size,omitempty"`
	Status      string `json:"status,omitempty"`
	actions     []string
	ifMatches   []string
	ifUnmatches []string
	invalids    []string
	IfClientIP  string `json:"ifclientip,omitempty"`
	IfProxy1IP  string `json:"ifproxy1ip,omitempty"`
	IfProxy2IP  string `json:"ifproxy2ip,omitempty"`
	IfTargetIP  string `json:"iftargetip,omitempty"`
	IfHostIP    string `json:"ifhostip,omitempty"`
	IfHost      string `json:"ifhost,omitempty"`
	IfAZ        string `json:"ifaz,omitempty"`
}

func (qs *QueryString) getValue(key string) (ret string) {
	switch key {
	case "cpu":
		ret = qs.CPU
	case "memory":
		ret = qs.Memory
	case "sleep":
		ret = qs.Sleep
	case "size":
		ret = qs.Size
	case "status":
		ret = qs.Status
	}
	return
}

func (qs *QueryString) setValue(key, value string) {
	switch key {
	case "cpu":
		qs.CPU = value
	case "memory":
		qs.Memory = value
	case "sleep":
		qs.Sleep = value
	case "size":
		qs.Size = value
	case "status":
		qs.Status = value
	case "ifclientip":
		qs.IfClientIP = value
	case "ifproxy1ip":
		qs.IfProxy1IP = value
	case "ifproxy2ip":
		qs.IfProxy2IP = value
	case "iftargetip":
		qs.IfTargetIP = value
	case "ifhostip":
		qs.IfHostIP = value
	case "ifhost":
		qs.IfHost = value
	case "ifaz":
		qs.IfAZ = value
	}
}

func newValidator() map[string]*regexp.Regexp {
	const (
		regexpPercent  = "^(100|[0-9]{1,2})$"
		regexpNumRange = "^([0-9]+)(?:-([0-9]+))?$"
		//regexpNumComma = "^([0-9]+)(?:,([0-9]+))*$" // 2個以上はFindStringSubmatchで取得不可のためmatchしたらstrings.Split
		regexpStatus   = "^(200|400|403|404|500|502|503|504)$"
		regexpHostname = "^([a-zA-Z0-9-.]+)$"
		regexpAZone    = "^([a-z]{2}-[a-z]+-[1-9][a-d])$"
		regexpIPv4     = "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
		regexpIPv6     = "^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$"
	)
	validator := map[string]*regexp.Regexp{}
	validator["cpu"] = regexp.MustCompile(regexpPercent)
	validator["memory"] = regexp.MustCompile(regexpPercent)
	validator["sleep"] = regexp.MustCompile(regexpNumRange)
	validator["size"] = regexp.MustCompile(regexpNumRange)
	validator["status"] = regexp.MustCompile(regexpStatus)
	validator["ifhost"] = regexp.MustCompile(regexpHostname)
	validator["ifaz"] = regexp.MustCompile(regexpAZone)
	validator["ifhostip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["iftargetip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["ifproxy1ip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["ifproxy2ip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["ifclientip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	return validator
}

func (reqInfo *RequestInfo) validateQueryString(mapQs map[string][]string) *QueryString {
	qs := &QueryString{}
	for key, value := range combineValues(mapQs) {
		if re, ok := store.validator[key]; ok {
			qs.setValue(key, value)
			if len(re.FindStringSubmatch(value)) > 0 {
				if strings.HasPrefix(key, "if") {
					if reqInfo.getActualValue(key) == value {
						qs.ifMatches = append(qs.ifMatches, key)
					} else {
						qs.ifUnmatches = append(qs.ifUnmatches, key)
					}
				} else {
					qs.actions = append(qs.actions, key)
				}
			} else {
				qs.invalids = append(qs.invalids, key)
			}
		}
	}
	return qs
}

func combineValues(input map[string][]string) map[string]string {
	output := map[string]string{}
	for key := range input {
		output[key] = strings.Join(input[key], ", ")
	}
	return output
}

func (reqInfo *RequestInfo) setIPAddresse(r *http.Request) {
	reqInfo.TargetIP = extractIPAddress(r.Host)
	xff := splitXFF(r.Header.Get("X-Forwarded-For"))
	if len(xff) == 0 {
		reqInfo.ClientIP = extractIPAddress(r.RemoteAddr)
	} else {
		reqInfo.ClientIP = xff[0]
	}
	if len(xff) >= 2 {
		reqInfo.Proxy1IP = xff[1]
	}
	if len(xff) >= 3 {
		reqInfo.Proxy2IP = xff[2]
	}
}

func extractIPAddress(ipport string) string {
	var ipaddr string
	if strings.HasPrefix(ipport, "[") {
		ipaddr = strings.Join(strings.Split(ipport, ":")[:len(strings.Split(ipport, ":"))-1], ":")
		ipaddr = strings.Trim(ipaddr, "[]")
	} else {
		ipaddr = strings.Split(ipport, ":")[0]
	}
	return ipaddr
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

func (qs *QueryString) evaluate(reqInfo *RequestInfo) *QueryString {
	actionQs := QueryString{}
	if len(qs.actions) == 0 {
		return &actionQs
	}
	for _, invalid := range qs.invalids {
		actionQs.setValue(invalid, "invalid")
	}
	for _, ifMatch := range qs.ifMatches {
		actionQs.setValue(ifMatch, "matched")
	}
	for _, ifUnmatch := range qs.ifUnmatches {
		actionQs.setValue(ifUnmatch, "unmatched")
	}
	// action evaluation
	for _, action := range qs.actions {
		actionQs.setValue(action, qs.getActionValue(action))
	}
	return &actionQs
}

func (qs *QueryString) needsAction() bool {
	if len(qs.actions) == 0 {
		return false
	}
	if len(qs.ifUnmatches) != 0 {
		return false
	}
	return true
}

func (qs *QueryString) getActionValue(key string) (ret string) {
	if key == "sleep" || key == "size" {
		values := strings.Split(qs.getValue(key), "-")
		if len(values) == 1 {
			ret = qs.getValue(key)
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
	} else {
		ret = qs.getValue(key)
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
	case "iftargetip":
		ret = reqInfo.TargetIP
	case "ifhostip":
		ret = store.host.IP
	case "ifhost":
		ret = store.host.Name
	case "ifaz":
		ret = store.host.AZ
	}
	return
}