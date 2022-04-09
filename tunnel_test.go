package proxima

import (
	"bufio"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTunnel(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ins, inc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, ins.Close())
			}()

			req, err := http.ReadRequest(bufio.NewReader(ins))
			assert.NoError(t, err)
			assert.Equal(t, http.MethodConnect, req.Method)

			resp := http.Response{
				StatusCode: http.StatusOK,
				Request:    req,
			}
			assert.NoError(t, resp.Write(ins))
		}()
		defer func() {
			assert.NoError(t, inc.Close())
		}()

		outs, outc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, outs.Close())
			}()

			resp, err := http.ReadResponse(bufio.NewReader(outs), nil)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}()
		defer func() {
			assert.NoError(t, outc.Close())
		}()

		assert.NoError(t, Tunnel(inc, outc, &net.TCPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 8080}, nil))
	})

	t.Run("inbound doesn't accept a CONNECT request", func(t *testing.T) {
		ins, inc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, ins.Close())
			}()
		}()
		defer func() {
			assert.NoError(t, inc.Close())
		}()

		assert.Error(t, Tunnel(inc, nil, &net.TCPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 8080}, nil))
	})

	t.Run("inbound doesn't reply to a CONNECT request", func(t *testing.T) {
		ins, inc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, ins.Close())
			}()

			req, err := http.ReadRequest(bufio.NewReader(ins))
			assert.NoError(t, err)
			assert.Equal(t, http.MethodConnect, req.Method)
		}()
		defer func() {
			assert.NoError(t, inc.Close())
		}()

		assert.Error(t, Tunnel(inc, nil, &net.TCPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 8080}, nil))
	})

	t.Run("inbound responds with a non-2XX status code", func(t *testing.T) {
		ins, inc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, ins.Close())
			}()

			req, err := http.ReadRequest(bufio.NewReader(ins))
			assert.NoError(t, err)
			assert.Equal(t, http.MethodConnect, req.Method)

			resp := http.Response{StatusCode: http.StatusInternalServerError}
			assert.NoError(t, resp.Write(ins))
		}()
		defer func() {
			assert.NoError(t, inc.Close())
		}()

		assert.Error(t, Tunnel(inc, nil, &net.TCPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 8080}, nil))
	})

	t.Run("outbound doesn't accept a response", func(t *testing.T) {
		ins, inc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, ins.Close())
			}()

			req, err := http.ReadRequest(bufio.NewReader(ins))
			assert.NoError(t, err)
			assert.Equal(t, http.MethodConnect, req.Method)

			resp := http.Response{
				StatusCode: http.StatusOK,
				Request:    req,
			}
			assert.NoError(t, resp.Write(ins))
		}()
		defer func() {
			assert.NoError(t, inc.Close())
		}()

		outs, outc := net.Pipe()
		go func() {
			defer func() {
				assert.NoError(t, outs.Close())
			}()
		}()
		defer func() {
			assert.NoError(t, outc.Close())
		}()

		assert.Error(t, Tunnel(inc, outc, &net.TCPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 8080}, nil))
	})
}
