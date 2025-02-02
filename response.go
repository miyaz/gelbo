package main

import (
	"fmt"
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

func writeResponse(w http.ResponseWriter, respSize int, respJSON []byte) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("expected http.ResponseWriter to be an http.Flusher")
	}
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	respJSON = append(respJSON, '\n', '\n')
	respJSONLength := len(respJSON)
	if respSize <= respJSONLength {
		fmt.Fprintf(w, "%s", respJSON[0:respSize])
	} else {
		fmt.Fprintf(w, "%s", respJSON)
		respSize = respSize - respJSONLength
		loopCount := respSize / loopUnit
		remainder := respSize % loopUnit
		for i := 0; i < loopCount; i++ {
			fmt.Fprintf(w, "%s\n", randBytes(randSrc, loopUnit-1))
		}
		if remainder != 0 {
			fmt.Fprintf(w, "%s", randBytes(randSrc, remainder))
		}
	}
	flusher.Flush()
	return nil
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
