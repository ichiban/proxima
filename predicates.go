package proxima

import (
	"context"
	"github.com/ichiban/prolog/engine"
	"github.com/jtacoma/uritemplates"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// URITemplate expands template with Key-Value in pairs list and unifies it with url.
func URITemplate(template, pairs, url engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	switch temp := env.Resolve(template).(type) {
	case engine.Variable:
		return engine.Error(engine.InstantiationError(template))
	case engine.Atom:
		t, err := uritemplates.Parse(string(temp))
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}

		values := map[string]interface{}{}
		if err := engine.EachList(pairs, func(elem engine.Term) error {
			if c, ok := env.Resolve(elem).(*engine.Compound); ok && c.Functor == "-" && len(c.Args) == 2 {
				if key, ok := env.Resolve(c.Args[0]).(engine.Atom); ok {
					values[string(key)] = env.Resolve(c.Args[1])
				}
			}
			return nil
		}, env); err != nil {
			return engine.Error(err)
		}

		ret, err := t.Expand(values)
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}

		return engine.Unify(url, engine.Atom(ret), k, env)
	default:
		return engine.Error(engine.TypeError("atom", template, "%s is not an atom.", template))
	}
}

// HostPort succeeds if hostPort is an atom consisting of an atom host, a colon, and an integer port.
func HostPort(hostPort, host, port engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	switch hp := env.Resolve(hostPort).(type) {
	case engine.Variable:
		switch h := env.Resolve(host).(type) {
		case engine.Variable:
			return engine.Error(engine.InstantiationError(host))
		case engine.Atom:
			switch p := env.Resolve(port).(type) {
			case engine.Variable:
				return engine.Error(engine.InstantiationError(host))
			case engine.Integer:
				return engine.Unify(hostPort, engine.Atom(net.JoinHostPort(string(h), strconv.Itoa(int(p)))), k, env)
			default:
				return engine.Error(engine.TypeError("integer", port, "%s is not an integer.", port))
			}
		default:
			return engine.Error(engine.TypeError("atom", host, "%s is not an atom.", host))
		}
	case engine.Atom:
		h, p, err := net.SplitHostPort(string(hp))
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}
		po, err := strconv.Atoi(p)
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}
		given := engine.Compound{Args: []engine.Term{host, port}}
		actual := engine.Compound{Args: []engine.Term{engine.Atom(h), engine.Integer(po)}}
		return engine.Unify(&given, &actual, k, env)
	default:
		return engine.Error(engine.TypeError("atom", hostPort, "%s is not an atom.", hostPort))
	}
}

func HTTPGet(uri, headers, proxy, resp engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	return engine.Delay(func(ctx context.Context) *engine.Promise {
		var req *http.Request
		switch u := env.Resolve(uri).(type) {
		case engine.Variable:
			return engine.Error(engine.InstantiationError(uri))
		case engine.Atom:
			var err error
			req, err = http.NewRequest(http.MethodGet, string(u), nil)
			if err != nil {
				return engine.Error(engine.SystemError(err))
			}
			h, err := termHeader(headers, env)
			if err != nil {
				return engine.Error(err)
			}
			for k, vs := range h {
				req.Header[k] = vs
			}
		default:
			return engine.Error(engine.TypeError("atom", uri, "%s is not an atom.", uri))
		}

		var c http.Client

		switch p := env.Resolve(proxy).(type) {
		case engine.Variable:
			return engine.Error(engine.InstantiationError(proxy))
		case engine.Atom:
			u, err := ParseURL(string(p))
			if err != nil {
				return engine.Error(engine.SystemError(err))
			}

			// copy from http.DefaultTransport except Proxy.
			c.Transport = &http.Transport{
				Proxy: http.ProxyURL(u),
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
		default:
			return engine.Error(engine.TypeError("atom", proxy, "%s is not an atom.", proxy))
		}

		res, err := c.Do(req)
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}

		b, err := io.ReadAll(res.Body)
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}

		if err := res.Body.Close(); err != nil {
			return engine.Error(engine.SystemError(err))
		}

		return engine.Unify(resp, &engine.Compound{
			Functor: "response",
			Args: []engine.Term{
				engine.Integer(res.StatusCode),
				headerTerm(res.Header),
				engine.Atom(b),
			},
		}, k, env)
	})
}

func termHeader(pairs engine.Term, env *engine.Env) (http.Header, error) {
	ret := http.Header{}
	if err := engine.EachList(pairs, func(elem engine.Term) error {
		switch e := env.Resolve(elem).(type) {
		case engine.Variable:
			return engine.InstantiationError(elem)
		case *engine.Compound:
			if e.Functor != "-" || len(e.Args) != 2 {
				break
			}

			k, ok := env.Resolve(e.Args[0]).(engine.Atom)
			if !ok {
				break
			}

			var vs []string
			if err := engine.Each(e.Args[1], func(elem engine.Term) error {
				v, ok := env.Resolve(elem).(engine.Atom)
				if !ok {
					return engine.TypeError("atom", elem, "%s is not an atom.", elem)
				}
				vs = append(vs, string(v))
				return nil
			}, env); err != nil {
				return err
			}
			ret[string(k)] = vs
		default:
			break
		}
		return engine.DomainError("header", elem, "%s is not a header.", elem)
	}, env); err != nil {
		return nil, err
	}
	return ret, nil
}

func headerTerm(h http.Header) engine.Term {
	pairs := make([]engine.Term, 0, len(h))
	for k, vs := range h {
		vals := make([]engine.Term, len(vs))
		for i, v := range vs {
			vals[i] = engine.Atom(v)
		}
		pairs = append(pairs, &engine.Compound{
			Functor: "-",
			Args: []engine.Term{
				engine.Atom(k),
				engine.List(vals...),
			},
		})
	}
	return engine.List(pairs...)
}

func Log(level, msg, pairs engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	return engine.Delay(func(ctx context.Context) *engine.Promise {
		log, ok := ctx.Value(logKey).(*zerolog.Logger)
		if !ok {
			return engine.Bool(false)
		}

		var e *zerolog.Event
		switch l := env.Resolve(level).(type) {
		case engine.Variable:
			return engine.Error(engine.InstantiationError(level))
		case engine.Atom:
			switch l {
			case "debug":
				e = log.Debug()
			case "info":
				e = log.Info()
			case "warn":
				e = log.Warn()
			case "error":
				e = log.Error()
			default:
				return engine.Error(engine.DomainError("log_level", level, "%s is neither debug, info, warn, nor error.", level))
			}
		default:
			return engine.Error(engine.TypeError("atom", level, "%s is not an atom.", level))
		}

		if err := engine.EachList(pairs, func(elem engine.Term) error {
			switch pair := env.Resolve(elem).(type) {
			case engine.Variable:
				return engine.InstantiationError(elem)
			case *engine.Compound:
				if pair.Functor != "-" || len(pair.Args) != 2 {
					break
				}

				switch k := env.Resolve(pair.Args[0]).(type) {
				case engine.Atom:
					switch v := env.Resolve(pair.Args[1]).(type) {
					case engine.Variable:
						return engine.InstantiationError(pair.Args[1])
					case engine.Atom:
						e = e.Str(string(k), string(v))
					case engine.Integer:
						e = e.Int64(string(k), int64(v))
					case engine.Float:
						e = e.Float64(string(k), float64(v))
					default:
						e = e.Str(string(k), v.String())
					}
					return nil
				}
			default:
				break
			}
			return engine.TypeError("pair", elem, "%s is not a pair.", elem)
		}, env); err != nil {
			return engine.Error(err)
		}

		switch m := env.Resolve(msg).(type) {
		case engine.Variable:
			return engine.Error(engine.InstantiationError(msg))
		case engine.Atom:
			e.Msg(string(m))
			return k(env)
		default:
			return engine.Error(engine.TypeError("atom", msg, "%s is not an atom.", msg))
		}
	})
}

func RequestCounter(val engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	return engine.Delay(func(ctx context.Context) *engine.Promise {
		id, ok := hlog.IDFromCtx(ctx)
		if !ok {
			return engine.Bool(false)
		}

		return engine.Unify(val, engine.Integer(id.Counter()), k, env)
	})
}
