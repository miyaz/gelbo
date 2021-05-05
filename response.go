package main

import (
	"bufio"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const (
	letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	loopUnit      = 100
)

func writeResponse(w http.ResponseWriter, respSize int, respJSON []byte) {
	fw := bufio.NewWriter(w)
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	respJSON = append(respJSON, '\n', '\n')
	respJSONLength := len(respJSON)
	if respSize <= respJSONLength {
		fw.Write(respJSON[0:respSize])
	} else {
		fw.Write(respJSON)
		respSize = respSize - respJSONLength
		loopCount := respSize / loopUnit
		remainder := respSize % loopUnit
		for i := 0; i < loopCount; i++ {
			fw.Write(randBytes(randSrc, loopUnit-1))
			fw.Write([]byte("\n"))
		}
		if remainder != 0 {
			fw.Write(randBytes(randSrc, remainder))
		}
	}

	err := fw.Flush()
	if err != nil {
		log.Fatalln(err)
	}
}

func randBytes(randSrc *rand.Rand, n int) []byte {
	b := make([]byte, n)
	// A randSrc.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, randSrc.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}
