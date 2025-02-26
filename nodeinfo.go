package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NodeInfo ... information of node
type NodeInfo struct {
	*sync.RWMutex
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`

	RequestCount  int64 `json:"request_count"`
	SentBytes     int64 `json:"sent_bytes"`
	ReceivedBytes int64 `json:"received_bytes"`

	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	ActiveConns int64   `json:"active_conns"`
	TotalConns  int64   `json:"total_conns"`

	ELBs map[string]*NodeInfo `json:"elbs,omitempty"`
}

// NewNodeInfo ... create node info instance
func NewNodeInfo() *NodeInfo {
	now := time.Now().UnixNano()
	_node := &NodeInfo{
		CreatedAt: now,
		UpdatedAt: now,
		ELBs:      make(map[string]*NodeInfo),
	}
	_node.RWMutex = &sync.RWMutex{}
	return _node
}
func (ni *NodeInfo) getUpdatedAt() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.UpdatedAt
}
func (ni *NodeInfo) reflectRequest(receivedBytes, sentBytes int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.ReceivedBytes += receivedBytes
	ni.SentBytes += sentBytes
	ni.RequestCount++
	ni.UpdatedAt = time.Now().UnixNano()
}
func (ni *NodeInfo) getClone() *NodeInfo {
	ni.RLock()
	defer ni.RUnlock()
	node := *ni
	return &node
}
func (ni *NodeInfo) getCreatedAt() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.CreatedAt
}
func (ni *NodeInfo) setCreatedAt(_time int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.CreatedAt = _time
}

func (ni *NodeInfo) addTotalConns(cnt int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.UpdatedAt = time.Now().UnixNano()
	ni.TotalConns += cnt
}
func (ni *NodeInfo) addActiveConns(cnt int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.UpdatedAt = time.Now().UnixNano()
	ni.ActiveConns += cnt
}

func monitorHandler(w http.ResponseWriter, r *http.Request) {
	var rawFlag bool
	qsMap := r.URL.Query()
	for key := range qsMap {
		if key == "raw" {
			rawFlag = true
			break
		}
	}
	updateNode()
	if rawFlag {
		fmt.Fprintf(w, "\n%s\n", getStoreNodeJSON())
	} else {
		fmt.Fprintf(w, "\n%s\n", easeReadJSON(getStoreNodeJSON()))
	}
}

func updateNode() {
	store.node.Lock()
	defer store.node.Unlock()
	store.node.CPU = store.resource.CPU.getCurrent()
	store.node.Memory = store.resource.Memory.getCurrent()
	store.node.ActiveConns = cw.getActiveConns()
	store.node.TotalConns = cw.getTotalConns()
	store.node.UpdatedAt = time.Now().UnixNano()
}

// ELBNode ... temp struct for json.MarshalIndent
type ELBNode struct {
	ELBs map[string]*NodeInfo `json:"elbs"`
}

func getStoreNodeJSON() []byte {
	store.RLock()
	defer store.RUnlock()
	storeNodeJSON, err := json.MarshalIndent(store.node, "", "  ")
	if err != nil {
		fmt.Printf("failed to json.MarshalIndent: %v", err)
		return []byte{}
	}
	return storeNodeJSON
}

var utimeRegexp = regexp.MustCompile(`_at": ([0-9]{19}),`)
var bytesRegexp = regexp.MustCompile(`_bytes": ([0-9]+),`)
var usageRegexp = regexp.MustCompile(`(cpu|memory)": ([0-9.]+),?`)

func easeReadJSON(inputJSON []byte) (readableJSON string) {
	// TODO: tuning replace speed
	buffer := bytes.NewBuffer(inputJSON)
	line, err := buffer.ReadString('\n')
	for err == nil {
		// unixtime -> rfc3339 format string
		matches := utimeRegexp.FindStringSubmatch(line)
		if len(matches) > 1 {
			line = strings.Replace(line, matches[1], `"`+easeReadUnixTime(matches[1])+`"`, 1)
		} else {
			// bytes -> human readable string
			matches = bytesRegexp.FindStringSubmatch(line)
			if len(matches) > 1 {
				line = strings.Replace(line, matches[1], `"`+easeReadBytes(matches[1])+`"`, 1)
			} else {
				// usageRate -> round decimal string
				matches = usageRegexp.FindStringSubmatch(line)
				if len(matches) > 1 {
					line = strings.Replace(line, matches[2], easeReadUsageRate(matches[2]), 1)
				}
			}
		}
		readableJSON += line
		line, err = buffer.ReadString('\n')
	}
	readableJSON += line
	return
}

func easeReadBytes(sb string) string {
	b, _ := strconv.Atoi(sb)
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func easeReadUnixTime(st string) string {
	t, _ := strconv.ParseInt(st, 10, 64)
	return time.Unix(0, t).Format(time.RFC3339)
}

func easeReadUsageRate(su string) string {
	u, _ := strconv.ParseFloat(su, 64)
	return fmt.Sprintf("%.1f", math.Round(u*10)/10)
}
