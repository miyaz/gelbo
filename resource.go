package main

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const tickerGlobal = 100
const statInterval = 500

func cpuControl(cpuTargetChan chan float64) {
	go procStatParser()
	go cpuStress(cpuTargetChan)
}

func cpuStress(cpuTargetChan chan float64) {
	started := false
	cpuUsage := 0.0
	t := time.NewTicker(time.Duration(tickerGlobal) * time.Millisecond)
	defer t.Stop()
	quit := make(chan bool)
	ratio := make(chan float64)
	for {
		select {
		case <-t.C:
			if started {
				curCPUUsage := store.resource.CPU.getCurrent()
				if curCPUUsage < cpuUsage {
					newRatio := 1 - (cpuUsage-curCPUUsage)/1000
					//newRatio := 0.99
					//fmt.Printf("%6.2f < %6.2f: %6.4f %3d\n", curCPUUsage, cpuUsage, newRatio, runtime.NumGoroutine())
					ratio <- newRatio
				} else {
					newRatio := 1 + (curCPUUsage-cpuUsage)/1000
					//newRatio := 1.01
					//fmt.Printf("%6.2f > %6.2f: %6.4f %3d\n", curCPUUsage, cpuUsage, newRatio, runtime.NumGoroutine())
					ratio <- newRatio
				}
			}
		case newCPUUsage := <-cpuTargetChan:
			if started {
				if newCPUUsage == 0 {
					started = false
					quit <- true
				}
			} else {
				started = true
				go cpuUsageController(ratio, quit)
			}
			if 0 <= newCPUUsage && newCPUUsage <= 100 {
				cpuUsage = newCPUUsage
			}
		}
	}
}

func procStatParser() {
	//t := time.NewTicker(time.Duration(tickerGlobal) * time.Millisecond)
	t := time.NewTicker(time.Duration(statInterval) * time.Millisecond)
	defer t.Stop()
	var prevIdleTime, prevTotalTime uint64
	for i := 0; ; i++ {
		select {
		case <-t.C: //タイマーイベント
			file, err := os.Open("/proc/stat")
			if err != nil {
				log.Fatal(err)
			}
			scanner := bufio.NewScanner(file)
			scanner.Scan()
			firstLine := scanner.Text()[5:] // get rid of cpu plus 2 spaces
			file.Close()
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
			split := strings.Fields(firstLine)
			idleTime, _ := strconv.ParseUint(split[3], 10, 64)
			totalTime := uint64(0)
			for _, s := range split {
				u, _ := strconv.ParseUint(s, 10, 64)
				totalTime += u
			}
			if i > 0 {
				deltaIdleTime := idleTime - prevIdleTime
				deltaTotalTime := totalTime - prevTotalTime
				cpuUsage := (1.0 - float64(deltaIdleTime)/float64(deltaTotalTime)) * 100.0
				store.resource.CPU.setCurrent(cpuUsage)
			}
			prevIdleTime = idleTime
			prevTotalTime = totalTime
		}
	}
}

func cpuUsageController(ratio chan float64, quit chan bool) {
	interval := float64(1000)
	prevInterval := interval
	t := time.NewTicker(time.Duration(interval) * time.Millisecond)

	for {
		select {
		case <-t.C:
			done := make(chan int)
			go placeLoad(done)
			go stopTimer(done)
		case newRatio := <-ratio:
			if prevInterval != newRatio*interval {
				interval *= newRatio
				prevInterval = interval
				if interval > 1.0 {
					t.Stop()
					t = time.NewTicker(time.Duration(interval) * time.Millisecond)
					//log.Println("ticker changing to " + fmt.Sprint(interval) + " ms")
				} else {
					//log.Println("ticker no changing, cause be less than 100.0 ms")
				}
			}
		case <-quit:
			t.Stop()
			log.Println("..ticker stopped!")
			return
		}
	}
}

func stopTimer(done chan int) {
	time.Sleep(10 * time.Millisecond)
	close(done)
}

func placeLoad(done chan int) {
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}
}
