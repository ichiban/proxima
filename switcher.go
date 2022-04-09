package proxima

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/ichiban/prolog"
	"github.com/ichiban/prolog/engine"
	"github.com/rs/zerolog/hlog"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	proxyAuthorization = "Proxy-Authorization"
	prefix             = "Basic "
	scheme             = "http://"
)

//go:embed predicates.pl
var predicates string

type Switcher struct {
	*prolog.Interpreter
}

func New(files []string) (*Switcher, error) {
	s := Switcher{
		Interpreter: prolog.New(nil, nil),
	}

	s.Register3("host_port", HostPort)
	s.Register3("uri_template", URITemplate)
	s.Register4("probe", Probe)
	s.Register3("log", Log)

	if err := s.Exec(predicates); err != nil {
		return nil, err
	}

	for _, file := range files {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		if err := s.Exec(string(b)); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

type contextKey struct{}

var LogKey contextKey

func (s *Switcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)

	if r.Method != http.MethodConnect {
		log.Error().Str("method", r.Method).Msg(http.StatusText(http.StatusMethodNotAllowed))
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	u, err := ParseURL(scheme + r.RequestURI)
	if err != nil {
		log.Err(err).Msg("url.Parse(RequestURI) failed")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	target, err := net.ResolveTCPAddr("tcp", u.Host)
	if err != nil {
		log.Err(err).Msg("net.ResolveTCPAddr() failed")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	opts, err := s.options(r)
	if err != nil {
		log.Err(err).Msg("s.options() failed")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, LogKey, log)

	sols, err := s.QueryContext(ctx, `tunnel(Proxy, ?).`, opts)
	if err != nil {
		log.Err(err).Msg("s.Query() failed")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = sols.Close()
	}()

	for sols.Next() {
		var s struct {
			Proxy string
		}
		if err := sols.Scan(&s); err != nil {
			log.Err(err).Msg("sols.Scan() failed")
			continue
		}

		log := log.With().Str("proxy", s.Proxy).Logger()

		u, err := url.Parse(scheme + s.Proxy)
		if err != nil {
			log.Err(err).Msg("url.Parse(s.Proxy) failed")
			continue
		}

		header := make(http.Header, len(r.Header)+1)
		for k, vs := range r.Header {
			header[k] = vs
		}
		if u.User != nil {
			header.Set(proxyAuthorization, prefix+base64.StdEncoding.EncodeToString([]byte(u.User.String())))
		}

		inbound, err := net.Dial("tcp", u.Host)
		if err != nil {
			log.Warn().Err(err).Msg("net.Dial() failed")
			continue
		}

		h, ok := w.(http.Hijacker)
		if !ok {
			continue
		}
		outbound, _, err := h.Hijack()
		if err != nil {
			log.Err(err).Msg("h.Hijack() failed")
			continue
		}

		log.Info().Msg("tunnel start")
		if err := Tunnel(inbound, outbound, target, r.Header); err != nil {
			log.Warn().Err(err).Msg("Tunnel() failed")
			continue
		}
		log.Info().Msg("tunnel finish")

		return
	}

	if err := sols.Err(); err != nil {
		log.Err(err).Msg("sols.Err() failed")
	}

	http.Error(w, "", http.StatusBadGateway)
	log.Info().Msg("no tunnels")
}

func (s *Switcher) options(r *http.Request) (engine.Term, error) {
	rid, _ := hlog.IDFromRequest(r)

	elems := []engine.Term{
		&engine.Compound{
			Functor: "rid",
			Args: []engine.Term{
				engine.Integer(rid.Counter()),
			},
		},
		&engine.Compound{
			Functor: "remote",
			Args: []engine.Term{
				engine.Atom(r.RemoteAddr),
			},
		},
		&engine.Compound{
			Functor: "target",
			Args: []engine.Term{
				engine.Atom(r.RequestURI),
			},
		},
	}

	auth := r.Header.Get(proxyAuthorization)
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return engine.List(elems...), nil
	}

	b, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return nil, err
	}

	i := strings.Index(string(b), ":")
	t, err := s.Parser(strings.NewReader(fmt.Sprintf("[%s].", string(b[:i]))), nil).Term()
	if err != nil {
		return nil, err
	}
	return engine.ListRest(t, elems...), nil
}

// ParseURL parses a URL. 'http://' scheme will be assumed if omitted.
func ParseURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = scheme + raw
	}
	return url.Parse(raw)
}
