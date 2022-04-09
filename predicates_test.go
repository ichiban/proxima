package proxima

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ichiban/prolog/engine"
	"github.com/stretchr/testify/assert"
)

func TestURITemplate(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		u := engine.NewVariable()
		ok, err := URITemplate(engine.Atom("{id}:{pass}@localhost:{port}"), engine.List(
			engine.Atom("-").Apply(engine.Atom("id"), engine.Atom("foo")),
			engine.Atom("-").Apply(engine.Atom("pass"), engine.Atom("bar")),
			engine.Atom("-").Apply(engine.Atom("port"), engine.Integer(8080)),
		), u, func(env *engine.Env) *engine.Promise {
			assert.Equal(t, engine.Atom("foo:bar@localhost:8080"), env.Resolve(u))
			return engine.Bool(true)
		}, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("template is a variable", func(t *testing.T) {
		_, err := URITemplate(engine.Variable("Temp"), engine.List(), engine.Variable("URL"), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.ErrInstantiation, err)
	})

	t.Run("template is not an atom", func(t *testing.T) {
		_, err := URITemplate(engine.Integer(0), engine.List(), engine.Variable("URL"), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.TypeErrorAtom(engine.Integer(0)), err)
	})

	t.Run("template is not a valid URI template", func(t *testing.T) {
		_, err := URITemplate(engine.Atom("{{}}"), engine.List(), engine.Variable("URL"), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.DomainError("uri_template", engine.Atom("{{}}")), err)
	})

	t.Run("pairs is not a proper list", func(t *testing.T) {
		_, err := URITemplate(engine.Atom(""), engine.Variable("Pairs"), engine.Variable("URL"), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.ErrInstantiation, err)
	})
}

func TestHostPort(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		t.Run("split", func(t *testing.T) {
			ok, err := HostPort(engine.Atom("localhost:8080"), engine.Atom("localhost"), engine.Integer(8080), engine.Success, nil).Force(context.Background())
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("concatenation", func(t *testing.T) {
			hp := engine.Variable("HostPort")
			ok, err := HostPort(hp, engine.Atom("localhost"), engine.Integer(8080), func(env *engine.Env) *engine.Promise {
				assert.Equal(t, engine.Atom("localhost:8080"), env.Resolve(hp))
				return engine.Bool(true)
			}, nil).Force(context.Background())
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("hostPort is a variable", func(t *testing.T) {
		t.Run("host is a variable", func(t *testing.T) {
			_, err := HostPort(engine.Variable("HostPort"), engine.Variable("Host"), engine.Integer(8080), engine.Success, nil).Force(context.Background())
			assert.Equal(t, engine.ErrInstantiation, err)
		})

		t.Run("host is not an atom", func(t *testing.T) {
			_, err := HostPort(engine.Variable("HostPort"), engine.Integer(0), engine.Integer(8080), engine.Success, nil).Force(context.Background())
			assert.Equal(t, engine.TypeErrorAtom(engine.Integer(0)), err)
		})

		t.Run("port is a variable", func(t *testing.T) {
			_, err := HostPort(engine.Variable("HostPort"), engine.Atom("localhost"), engine.Variable("Port"), engine.Success, nil).Force(context.Background())
			assert.Equal(t, engine.ErrInstantiation, err)
		})

		t.Run("port is not an integer", func(t *testing.T) {
			_, err := HostPort(engine.Variable("HostPort"), engine.Atom("localhost"), engine.Float(8080), engine.Success, nil).Force(context.Background())
			assert.Equal(t, engine.TypeErrorInteger(engine.Float(8080)), err)
		})
	})

	t.Run("hostPort is an atom", func(t *testing.T) {
		t.Run("hostPort is not a form of host:port", func(t *testing.T) {
			_, err := HostPort(engine.Atom("foo"), engine.Atom("localhost"), engine.Integer(8080), engine.Success, nil).Force(context.Background())
			assert.Equal(t, engine.DomainError("host_port", engine.Atom("foo")), err)
		})
	})

	t.Run("hostPort is not an atom", func(t *testing.T) {
		_, err := HostPort(engine.Integer(0), engine.Atom("localhost"), engine.Integer(8080), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.TypeErrorAtom(engine.Integer(0)), err)
	})
}

func TestProbe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		clientDo = func(c *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
		defer func() {
			clientDo = (*http.Client).Do
		}()

		ok, err := Probe(engine.Atom("localhost:8080"), engine.Atom("https://example.com/ok"), engine.List(engine.Atom("-").Apply(engine.Atom("User-Agent"), engine.List(engine.Atom("proxima")))), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("proxy is a variable", func(t *testing.T) {
		_, err := Probe(engine.Variable("Proxy"), engine.Atom("https://example.com/ok"), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.ErrInstantiation, err)
	})

	t.Run("proxy is not an atom", func(t *testing.T) {
		_, err := Probe(engine.Integer(0), engine.Atom("https://example.com/ok"), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.TypeErrorAtom(engine.Integer(0)), err)
	})

	t.Run("proxy is not a URL", func(t *testing.T) {
		_, err := Probe(engine.Atom(" "), engine.Atom("https://example.com/ok"), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.DomainError("url", engine.Atom(" ")), err)
	})

	t.Run("target is a variable", func(t *testing.T) {
		_, err := Probe(engine.Atom("localhost:8080"), engine.Variable("Target"), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.ErrInstantiation, err)
	})

	t.Run("target is not an atom", func(t *testing.T) {
		_, err := Probe(engine.Atom("localhost:8080"), engine.Integer(0), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.TypeErrorAtom(engine.Integer(0)), err)
	})

	t.Run("target is not a URL", func(t *testing.T) {
		_, err := Probe(engine.Atom("localhost:8080"), engine.Atom("http:// "), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.DomainError("url", engine.Atom("http:// ")), err)
	})

	t.Run("request failed", func(t *testing.T) {
		clientDo = func(c *http.Client, req *http.Request) (*http.Response, error) {
			return nil, errors.New("failed")
		}
		defer func() {
			clientDo = (*http.Client).Do
		}()

		ok, err := Probe(engine.Atom("localhost:8080"), engine.Atom("https://example.com/ok"), engine.List(), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("options is not a proper list", func(t *testing.T) {
		_, err := Probe(engine.Atom("localhost:8080"), engine.Atom("https://example.com/ok"), engine.ListRest(engine.Variable("Rest")), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.ErrInstantiation, err)
	})

	t.Run("option is a variable", func(t *testing.T) {
		_, err := Probe(engine.Atom("localhost:8080"), engine.Atom("https://example.com/ok"), engine.List(engine.Variable("Option")), engine.Integer(200), engine.Success, nil).Force(context.Background())
		assert.Equal(t, engine.ErrInstantiation, err)
	})
}
