package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// NodeInfo ... information of node
type NodeInfo struct {
	*sync.RWMutex
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
	Reachable bool  `json:"reachable"`

	Count       int64   `json:"count"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	Bytes       int64   `json:"bytes"`
	ActiveConns int64   `json:"active_conns"`
	TotalConns  int64   `json:"total_conns"`
}

// NewNodeInfo ... create node info instance
func NewNodeInfo() *NodeInfo {
	now := time.Now().UnixNano()
	_node := &NodeInfo{
		CreatedAt: now,
		UpdatedAt: now,
		Reachable: true,
	}
	_node.RWMutex = &sync.RWMutex{}
	return _node
}
func (ni *NodeInfo) getUpdatedAt() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.UpdatedAt
}
func (ni *NodeInfo) setUpdatedAt(_time int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.UpdatedAt = _time
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
func (ni *NodeInfo) reflectRequest(bytes int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.Bytes += bytes
	ni.Count++
	ni.UpdatedAt = time.Now().UnixNano()
}
func (ni *NodeInfo) getClone() *NodeInfo {
	ni.RLock()
	defer ni.RUnlock()
	node := *ni
	return &node
}

func (ni *NodeInfo) setCount(cnt int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.Count = cnt
}
func (ni *NodeInfo) countUp() {
	ni.Lock()
	defer ni.Unlock()
	ni.Count++
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

func monitorHandler(w http.ResponseWriter, r *http.Request) {
	syncer.RLock()
	defer syncer.RUnlock()
	syncerJSON, _ := json.MarshalIndent(syncer, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(syncerJSON))
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
				syncer.Nodes[nodeIP].setCount(inNode.getCount())
				syncer.Nodes[nodeIP].setCreatedAt(inNode.getCreatedAt())
				syncer.Nodes[nodeIP].setUpdatedAt(inNode.getUpdatedAt())
				syncer.Nodes[nodeIP].setReachable(inNode.isReachable())
			}
		} else {
			syncer.Lock()
			inNode.setReachable(true)
			syncer.Nodes[nodeIP] = inNode
			syncer.Unlock()
		}
	}
}

func getSyncerJSON() []byte {
	updateSyncer()
	syncer.RLock()
	defer syncer.RUnlock()
	syncerJSON, err := json.MarshalIndent(syncer, "", "  ")
	if err != nil {
		fmt.Printf("failed to json.MarshalIndent: %v", err)
		return []byte{}
	}
	return syncerJSON
}
func updateSyncer() {
	syncer.setNow()
	store.node.setNow()
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
	if merge {
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
