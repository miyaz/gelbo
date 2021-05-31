package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	CreatedAt   int64 `json:"created_at"`
	UpdatedAt   int64 `json:"updated_at"`
	Reachable   bool  `json:"reachable"`
	SyncerCount int64 `json:"syncer_count,omitempty"`

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
		Reachable: true,
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
func (ni *NodeInfo) getSyncerCount() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.SyncerCount
}
func (ni *NodeInfo) updateConns() {
	ni.Lock()
	defer ni.Unlock()
	ni.ActiveConns = cw.getActiveConns()
	ni.TotalConns = cw.getTotalConns()
}
func (ni *NodeInfo) updateResources() {
	ni.Lock()
	defer ni.Unlock()
	ni.CPU = store.resource.CPU.getCurrent()
	ni.Memory = store.resource.Memory.getCurrent()
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
func (ni *NodeInfo) countUp() {
	ni.Lock()
	defer ni.Unlock()
	ni.SyncerCount++
	ni.UpdatedAt = time.Now().UnixNano()
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
func (ni *NodeInfo) setNow() {
	ni.Lock()
	defer ni.Unlock()
	ni.UpdatedAt = time.Now().UnixNano()
}
func (ni *NodeInfo) addTotalConnsELB(remoteAddr string, cnt int64) {
	ni.Lock()
	defer ni.Unlock()
	now := time.Now().UnixNano()
	if _, ok := ni.ELBs[remoteAddr]; !ok {
		ni.ELBs[remoteAddr] = &NodeInfo{}
		ni.ELBs[remoteAddr].CreatedAt = now
	}
	ni.ELBs[remoteAddr].UpdatedAt = now
	ni.ELBs[remoteAddr].TotalConns += cnt
}
func (ni *NodeInfo) addActiveConnsELB(remoteAddr string, cnt int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.ELBs[remoteAddr].UpdatedAt = time.Now().UnixNano()
	ni.ELBs[remoteAddr].TotalConns += cnt
}

// Syncer ... Latest Data for Syncer
type Syncer struct {
	*sync.RWMutex
	SyncedAt int64                `json:"synced_at"`
	Nodes    map[string]*NodeInfo `json:"nodes"`
}

func (s *Syncer) setNow() {
	s.Lock()
	defer s.Unlock()
	s.SyncedAt = time.Now().UnixNano()
}
func (s *Syncer) getSyncedAt() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.SyncedAt
}

var syncer = Syncer{&sync.RWMutex{}, time.Now().UnixNano(), map[string]*NodeInfo{}}

func elbStatsHandler(w http.ResponseWriter, r *http.Request) {
	var rawFlg bool
	qsMap := r.URL.Query()
	for key := range qsMap {
		if key == "raw" {
			rawFlg = true
			break
		}
	}
	updateSyncer()
	if rawFlg {
		fmt.Fprintf(w, "\n%s\n", getSyncerELBJSON())
	} else {
		fmt.Fprintf(w, "\n%s\n", easeReadJSON(getSyncerELBJSON()))
	}
}

func monitorHandler(w http.ResponseWriter, r *http.Request) {
	var rawFlg bool
	qsMap := r.URL.Query()
	for key := range qsMap {
		if key == "raw" {
			rawFlg = true
			break
		}
	}
	updateSyncer()
	if rawFlg {
		fmt.Fprintf(w, "\n%s\n", getSyncerJSON())
	} else {
		fmt.Fprintf(w, "\n%s\n", easeReadJSON(getSyncerJSON()))
	}
}

func syncerHandler(w http.ResponseWriter, r *http.Request) {
	store.node.countUp()
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusBadRequest)
	case http.MethodPost:
		body := r.Body
		defer body.Close()
		buf := new(bytes.Buffer)
		io.Copy(buf, body)
		//wkSyncer := Syncer{&sync.RWMutex{}, time.Now().UnixNano(), map[string]NodeInfo{}}
		wkSyncer := Syncer{} // RWMutex や map にアクセスしなければ初期化は不要
		if err := json.Unmarshal(buf.Bytes(), &wkSyncer); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Bad Request\n")
			fmt.Printf("failed to json.MarshalIndent: %v", err)
		} else {
			mergeSyncer(&wkSyncer)
			updateSyncer()
			fmt.Fprintln(w, string(getSyncerJSON()))
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "Method not allowed.\n")
	}
}

func mergeSyncer(inSyncer *Syncer) {
	if inSyncer == nil || inSyncer.Nodes == nil {
		return
	}
	for nodeIP, inNode := range inSyncer.Nodes {
		if nodeIP == store.host.IP {
			continue
		}
		if inNode.RWMutex == nil {
			inNode.RWMutex = &sync.RWMutex{}
		}
		if node, ok := syncer.Nodes[nodeIP]; ok {
			if node.getUpdatedAt() < inNode.getUpdatedAt() {
				if node.getSyncerCount() > inNode.getSyncerCount() {
					// detect reboot process
					syncer.Lock()
					prevNodeIP := nodeIP + "_retired_" + strconv.FormatInt(node.getCreatedAt(), 10)
					syncer.Nodes[prevNodeIP] = syncer.Nodes[nodeIP]
					syncer.Nodes[prevNodeIP].setReachable(false)
					syncer.Unlock()
				}
				syncer.Lock()
				syncer.Nodes[nodeIP] = inNode
				syncer.Unlock()
			}
		} else {
			syncer.Lock()
			if strings.Index(nodeIP, "_") == -1 {
				// runnint process, not retired
				inNode.setReachable(true)
			}
			syncer.Nodes[nodeIP] = inNode
			syncer.Unlock()
		}
	}
}

func updateSyncer() {
	store.node.updateResources()
	store.node.updateConns()
	store.node.setNow()
	syncer.setNow()
	syncer.Lock()
	defer syncer.Unlock()
	syncer.Nodes[store.host.IP] = store.node.getClone()
}

func loopSyncer() {
	sleep := 500
	ticker := time.NewTicker(time.Duration(sleep) * time.Millisecond)
	defer ticker.Stop()
	for {
		//time.Sleep(time.Duration(sleep) * time.Millisecond)
		select {
		case <-ticker.C:
			syncer.RLock()
			now := time.Now().UnixNano()
			minUpdatedAt := now
			destIP := store.host.IP
			for nodeIP := range syncer.Nodes {
				curUpdatedAt := syncer.Nodes[nodeIP].getUpdatedAt()
				if minUpdatedAt > curUpdatedAt && syncer.Nodes[nodeIP].isReachable() {
					minUpdatedAt = curUpdatedAt
					destIP = nodeIP
				}
			}
			syncer.RUnlock()
			if destIP == store.host.IP {
				continue
			}
			delta := (float64)(now-minUpdatedAt) / 1000000000
			syncer.Nodes[destIP].setReachable(execSyncer("http://"+destIP+":"+strconv.Itoa(syncerPort)+"/syncer/", true))
			fmt.Println(string(getSyncerJSON()))
			fmt.Printf("ip %s : %d - %d = %d (%f sec)\n", destIP, now, minUpdatedAt, now-minUpdatedAt, delta)
		}
	}
}

func initSyncer() {
	nodes, _ := getEC2IPList()
	reachableNodes := getReachableNodeList(nodes)
	syncer.Lock()
	defer syncer.Unlock()
	for i := 0; i < len(reachableNodes); i++ {
		syncer.Nodes[reachableNodes[i]] = NewNodeInfo()
	}
}

func execSyncer(url string, merge bool) bool {
	c := &http.Client{
		Timeout: 500 * time.Millisecond,
	}
	updateSyncer()
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(getSyncerJSON()))
	if err != nil {
		fmt.Printf("failed to http.NewRequest: %v", err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	resp, err := c.Do(req)
	if err != nil {
		//fmt.Printf("failed to c.Do: %v", err)
		return false
	}
	defer resp.Body.Close()

	byteArray, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("failed to ioutil.ReadAll: %v", err)
	}
	wkSyncer := Syncer{} // RWMutex や map にアクセスしなければ初期化は不要
	if err := json.Unmarshal(byteArray, &wkSyncer); err != nil {
		fmt.Printf("failed to json.Unmarshal: %v", err)
	}
	if merge && wkSyncer.SyncedAt > 0 {
		mergeSyncer(&wkSyncer)
	}
	return true
}

func getReachableNodeList(nodes []string) (reachableNodes []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	limiter := make(chan struct{}, 10)
	for i := 0; i < len(nodes); i++ {
		nodeIP := nodes[i]
		wg.Add(1)
		go func() {
			limiter <- struct{}{}
			defer wg.Done()
			reachable := execSyncer("http://"+nodeIP+":"+strconv.Itoa(syncerPort)+"/syncer/", false)
			<-limiter
			mu.Lock()
			defer mu.Unlock()
			if reachable {
				reachableNodes = append(reachableNodes, nodeIP)
			}
		}()
	}
	wg.Wait()
	return
}

// ELBNode ... temp struct for json.MarshalIndent
type ELBNode struct {
	ELBs map[string]*NodeInfo `json:"elbs"`
}

func getSyncerELBJSON() []byte {
	syncer.RLock()
	defer syncer.RUnlock()
	elbNodes := map[string]*NodeInfo{}
	for ip, node := range syncer.Nodes {
		for elbIP, elbNode := range node.ELBs {
			if _, ok := elbNodes[elbIP]; !ok {
				elbNodes[elbIP] = &NodeInfo{}
			}
			if elbNode.CreatedAt != 0 && elbNode.CreatedAt < elbNodes[elbIP].CreatedAt {
				elbNodes[elbIP].CreatedAt = elbNode.CreatedAt
			}
			if elbNode.UpdatedAt > elbNodes[elbIP].UpdatedAt {
				elbNodes[elbIP].UpdatedAt = elbNode.UpdatedAt
			}
			elbNodes[elbIP].RequestCount += elbNode.RequestCount
			elbNodes[elbIP].SentBytes += elbNode.SentBytes
			elbNodes[elbIP].ReceivedBytes += elbNode.ReceivedBytes
			// exclude if retired node
			if strings.Index(ip, "_") == -1 {
				elbNodes[elbIP].ActiveConns += elbNode.ActiveConns
				elbNodes[elbIP].TotalConns += elbNode.TotalConns
			}
		}
	}
	elbsJSON, err := json.MarshalIndent(ELBNode{ELBs: elbNodes}, "", "  ")
	if err != nil {
		fmt.Printf("failed to json.MarshalIndent: %v", err)
		return []byte{}
	}
	return elbsJSON
}

func getSyncerJSON() []byte {
	syncer.RLock()
	defer syncer.RUnlock()
	syncerJSON, err := json.MarshalIndent(syncer, "", "  ")
	if err != nil {
		fmt.Printf("failed to json.MarshalIndent: %v", err)
		return []byte{}
	}
	return syncerJSON
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
			// unixtime -> rfc3339 format string
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
