package lb

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gopyai/go-ws"
)

var (
	DEBUG bool
)

var (
	errSetup        = errors.New("Error LoadBalancer setup")
	errNotAvailable = errors.New("All apps are down, wait for a while")
)

func LoadBalancer(
	wUrl []string,
	strip string,
	breakerLimit int,
	breakerSecs float64) http.Handler {

	if len(wUrl) == 0 || breakerLimit < 1 || breakerSecs <= 0 {
		panic(errSetup)
	}

	var wLck sync.Mutex
	wIdx := -1
	wErrCnt := make([]int, len(wUrl))
	wWakeUp := make([]time.Time, len(wUrl))
	wLatest := make([]time.Time, len(wUrl))

	return ws2.Handler(func(
		method, uri, contentType string, inHeader map[string]string, in []byte) (
		outHeader map[string]string, out []byte, code int, err error) {

		now := time.Now()

		// Find the right worker to handle the request
		wLck.Lock()
		for cnt := 0; ; cnt++ {
			if cnt > len(wUrl) {
				// All workers are not available
				wLck.Unlock()
				return nil, nil, http.StatusInternalServerError, errNotAvailable
			}

			// Next worker
			wIdx++
			if wIdx >= len(wUrl) {
				wIdx = 0
			}

			if now.Sub(wLatest[wIdx]).Seconds() >= breakerSecs {
				// Reset error counter
				wErrCnt[wIdx] = 0
				//wWakeUp[wIdx] = now
			}

			if now.After(wWakeUp[wIdx]) {
				// This worker is available
				break
			}
		}
		// Compose the URL
		if strings.HasPrefix(uri, strip) {
			uri = uri[len(strip):]
		}
		u := "http://" + wUrl[wIdx] + uri
		wLatest[wIdx] = now
		wLck.Unlock()

		// Call the URL
		h, o, c, e := ws2.Call(
			method, u, contentType,
			inHeader, in,
			0)
		if DEBUG {
			msg := fmt.Sprintf("### REQUEST ###\n%s %s %s\nHeader: %s\nBody: %s\n### RESPONSE ###\nHeader: %s\nResponse: %s\nHTTP Status: %d",
				method, u, contentType, inHeader, in,
				h, o, c)
			log.Println(msg)
			if e != nil {
				fmt.Println(e)
			}
		}

		if e != nil || c >= 500 {
			// Call error
			wLck.Lock()
			defer wLck.Unlock()
			wErrCnt[wIdx]++
			if wErrCnt[wIdx] >= breakerLimit {
				// Number of error is more than the limit for breaker to active
				wWakeUp[wIdx] = now.Add(time.Duration(float64(time.Second) * breakerSecs))
				wErrCnt[wIdx] = 0
			}
		}
		return h, o, c, e
	})
}
