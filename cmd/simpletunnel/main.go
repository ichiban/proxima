package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "port number")
	flag.Parse()

	s := http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect {
				http.Error(w, "", http.StatusMethodNotAllowed)
				return
			}

			log.Printf("CONNECT %s", r.URL.Host)

			_, _ = io.Copy(ioutil.Discard, r.Body)
			_ = r.Body.Close()

			inbound, err := net.Dial("tcp", r.URL.Host)
			if err != nil {
				http.Error(w, "", http.StatusBadGateway)
				return
			}

			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusOK)
			h := w.(http.Hijacker)
			outbound, _, err := h.Hijack()
			if err != nil {
				panic(err)
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer inbound.Close()

				_, _ = io.Copy(inbound, outbound)
			}()
			wg.Add(1)
			go func() {
				defer wg.Done()

				_, _ = io.Copy(outbound, inbound)
			}()
			wg.Wait()
		}),
	}

	err := s.ListenAndServe()
	switch err {
	case nil, http.ErrServerClosed:
	default:
		panic(err)
	}

	if err := s.Shutdown(context.Background()); err != nil {
		panic(err)
	}
}
