package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	lb "github.com/pknrj/SimpleLoadBalancer/internals/backend"
)

const (
	Attempts int = iota
	Retry
)

var loadServer lb.BackendPool

// GetAttemptsFromContext returns the attempts for request

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetRetryFromContext returns the retries for request

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

func serveRequest(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := loadServer.GetNextServer()
	if peer != nil {
		peer.RvsProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}


func healthCheck() {
	t := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-t.C:
			log.Println("Health check started !!!!! ")
			loadServer.HealthCheck()
			log.Println("Health check completed !!!!!!")
		}
	}
}


func main(){

	serverList := [] string {
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	for _ , val := range serverList {

		serverUrl , err := url.Parse(val)
		if err != nil {
			log.Fatal(err)
		}

		rproxy := httputil.NewSingleHostReverseProxy(serverUrl)
		rproxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error){
			log.Printf("[%s] %s\n", serverUrl.Host, err.Error())
			retries := GetRetryFromContext(r)

			if retries < 3 {
				select {
					case <- time.After(10 * time.Millisecond) : 
						ctx := context.WithValue(r.Context(), Retry, retries+1)
						rproxy.ServeHTTP(w, r.WithContext(ctx))
				}
				return
			}

			loadServer.SetBackendStatus(serverUrl , false)
			attempts := GetAttemptsFromContext(r)

			log.Printf("%s(%s) Attempting retry %d\n", r.RemoteAddr, r.URL.Path, attempts)

			ctx := context.WithValue(r.Context(), Attempts, attempts+1)

			serveRequest(w, r.WithContext(ctx))

		}

		loadServer.AppendBackend(&lb.Backend{
			URL: serverUrl,
			Alive: true,
			RvsProxy: rproxy,
		})

		log.Printf("Server added : %s\n" , serverUrl)
	}

	httpServer := http.Server {
		Addr: fmt.Sprintf(":%d", 8080),
		Handler: http.HandlerFunc(serveRequest),
	}

	go healthCheck()


	log.Printf("Load balancer server started at %d" , 8080)
	if err := httpServer.ListenAndServe() ; err != nil {
		log.Fatal(err)
	}

}