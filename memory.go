package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const memoryAllocTicker = 1000
const memoryCheckInterval = 500

func memoryControl(memoryTargetChan chan float64) {
	go procMeminfoParser()
	go memoryAllocate(memoryTargetChan)
}

func memoryAllocate(memoryTargetChan chan float64) {
	memTotal, err := getMemoryTotal()
	if err != nil {
		log.Fatal(err)
	}
	allocUnit := (int)(memTotal / 100)
	allocData := []string{}
	memoryUsage := 0.0
	t := time.NewTicker(time.Duration(memoryAllocTicker) * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			curMemoryUsage := store.resource.Memory.getCurrent()
			if curMemoryUsage < memoryUsage {
				allocData = append(allocData, allocMemory(allocUnit))
			} else {
				if len(allocData) != 0 {
					allocData = allocData[1:]
					runtime.GC()
				}
			}
		case newMemoryUsage := <-memoryTargetChan:
			if 0 <= newMemoryUsage && newMemoryUsage <= 100 {
				memoryUsage = newMemoryUsage
			}
		}
	}
}

func getMemoryTotal() (int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.IndexRune(line, ':')
		if i < 0 {
			continue
		}
		if line[:i] == "MemTotal" {
			val := strings.TrimSpace(strings.TrimRight(line[i+1:], "kB"))
			if v, err := strconv.Atoi(val); err == nil {
				return v * 1024, nil
			}
		}
	}
	file.Close()
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return 0, fmt.Errorf("MemTotal not found")
}

// Stats represents memory statistics for linux
type Stats struct {
	Total, Used, Buffers, Cached, Free, Available, Active, Inactive,
	SwapTotal, SwapUsed, SwapCached, SwapFree float64
}

func procMeminfoParser() {
	t := time.NewTicker(time.Duration(memoryCheckInterval) * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			file, err := os.Open("/proc/meminfo")
			if err != nil {
				log.Fatal(err)
			}
			scanner := bufio.NewScanner(file)
			var memory Stats
			memStats := map[string]*float64{
				"MemTotal":     &memory.Total,
				"MemFree":      &memory.Free,
				"MemAvailable": &memory.Available,
				"Buffers":      &memory.Buffers,
				"Cached":       &memory.Cached,
				"Active":       &memory.Active,
				"Inactive":     &memory.Inactive,
				"SwapCached":   &memory.SwapCached,
				"SwapTotal":    &memory.SwapTotal,
				"SwapFree":     &memory.SwapFree,
			}
			for scanner.Scan() {
				line := scanner.Text()
				i := strings.IndexRune(line, ':')
				if i < 0 {
					continue
				}
				fld := line[:i]
				if ptr := memStats[fld]; ptr != nil {
					val := strings.TrimSpace(strings.TrimRight(line[i+1:], "kB"))
					if v, err := strconv.ParseFloat(val, 64); err == nil {
						*ptr = v * 1024
					}
				}
			}
			file.Close()
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}

			memory.SwapUsed = memory.SwapTotal - memory.SwapFree
			memory.Used = memory.Total - memory.Available
			memoryUsage := memory.Used * 100.0 / memory.Total
			store.resource.Memory.setCurrent(memoryUsage)
		}
	}
}

func freeMemory(done chan int) {
	close(done)
	runtime.GC()
}

func allocMemory(size int) string {
	buf := new(bytes.Buffer)
	str := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ%#"
	for i := 0; i < size/len(str); i++ {
		buf.WriteString(str)
	}
	return buf.String()
}
