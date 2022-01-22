package proxima

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
)

// Tunnel connects inbound and outbound connections by making a CONNECT request to inbound.
func Tunnel(inbound, outbound net.Conn, target net.Addr, header http.Header) (func(), error) {
	req := http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Host: target.String(),
		},
		Header: header,
	}

	if err := req.Write(inbound); err != nil {
		return nil, err
	}

	br := bufio.NewReader(inbound)
	resp, err := http.ReadResponse(br, &req)
	if err != nil {
		return nil, err
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("status is not 2XX: %s", resp.Status)
	}

	if err := resp.Write(outbound); err != nil {
		return nil, err
	}

	return func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				_ = inbound.Close()
			}()
			_, _ = io.Copy(inbound, outbound)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				_ = outbound.Close()
			}()
			_, _ = io.Copy(outbound, inbound)
		}()
		wg.Wait()
	}, nil
}
