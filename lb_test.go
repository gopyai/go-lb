package lb_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gopyai/lb"
	"github.com/gopyai/ws2"
)

var (
	errTesting = errors.New("Error")
	willError  bool
)

func Example() {
	go runSvr()
	time.Sleep(time.Millisecond) // Allow the server to start for a while

	fmt.Println("### Scenario: All apps are OK ###")
	runCli(4, time.Millisecond*10)

	fmt.Println("### Scenario: One app is error and circuit breaker will be activated ###")
	willError = true
	runCli(6, time.Millisecond*10)

	time.Sleep(time.Millisecond * 30)

	fmt.Println("### Scenario: One app is error and circuit breaker will not be activated because of long delay ###")
	willError = true
	runCli(6, time.Millisecond*20)

	// Output:
	// ### Scenario: All apps are OK ###
	// >> app1
	// Result: 200 <nil>
	// >> app2
	// Result: 200 <nil>
	// >> app1
	// Result: 200 <nil>
	// >> app2
	// Result: 200 <nil>
	// ### Scenario: One app is error and circuit breaker will be activated ###
	// >> app1
	// Result: 200 <nil>
	// >> app2
	// Result: 500 <nil>
	// >> app1
	// Result: 200 <nil>
	// >> app2
	// Result: 500 <nil>
	// >> app1
	// Result: 200 <nil>
	// >> app1
	// Result: 200 <nil>
	// ### Scenario: One app is error and circuit breaker will not be activated because of long delay ###
	// >> app2
	// Result: 500 <nil>
	// >> app1
	// Result: 200 <nil>
	// >> app2
	// Result: 500 <nil>
	// >> app1
	// Result: 200 <nil>
	// >> app2
	// Result: 500 <nil>
	// >> app1
	// Result: 200 <nil>
}

func app1(
	method, uri, contentType string, inHeader map[string]string, in []byte) (
	outHeader map[string]string, out []byte, code int, err error) {

	fmt.Println(">> app1")
	return nil, nil, 200, nil
}

func app2(
	method, uri, contentType string, inHeader map[string]string, in []byte) (
	outHeader map[string]string, out []byte, code int, err error) {

	fmt.Println(">> app2")
	if willError {
		return nil, nil, http.StatusInternalServerError, errTesting
	}
	return nil, nil, 200, nil
}

func runSvr() {
	mux := http.NewServeMux()

	mux.Handle("/app1/", ws2.Handler(app1))
	mux.Handle("/app2/", ws2.Handler(app2))
	mux.Handle("/test/", lb.LoadBalancer([]string{
		"localhost:8080/app1/",
		"localhost:8080/app2/",
	}, "/test/", 2, 0.03))

	if e := http.ListenAndServe(":8080", mux); e != nil {
		fmt.Println("Error:", e)
		os.Exit(1)
	}
}

func runCli(n int, sleep time.Duration) {
	for i := 0; i < n; i++ {
		_, _, c, e := ws2.Call(
			"POST",
			"http://localhost:8080/test/halo",
			"application/json",
			nil,
			nil,
			0,
		)
		fmt.Println("Result:", c, e)
		time.Sleep(sleep)
	}
}
