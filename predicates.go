package proxima

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/ichiban/prolog/engine"
	"github.com/jtacoma/uritemplates"
	"github.com/rs/zerolog"
)

// URITemplate expands template with Key-Value in pairs list and unifies it with url.
func URITemplate(template, pairs, url engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	switch temp := env.Resolve(template).(type) {
	case engine.Variable:
		return engine.Error(engine.ErrInstantiation)
	case engine.Atom:
		t, err := uritemplates.Parse(string(temp))
		if err != nil {
			return engine.Error(engine.DomainError("uri_template", temp))
		}

		values := map[string]interface{}{}
		iter := engine.ListIterator{List: pairs, Env: env}
		for iter.Next() {
			if c, ok := env.Resolve(iter.Current()).(*engine.Compound); ok && c.Functor == "-" && len(c.Args) == 2 {
				if key, ok := env.Resolve(c.Args[0]).(engine.Atom); ok {
					values[string(key)] = env.Resolve(c.Args[1])
				}
			}
		}
		if err := iter.Err(); err != nil {
			return engine.Error(err)
		}

		ret, _ := t.Expand(values)

		return engine.Unify(url, engine.Atom(ret), k, env)
	default:
		return engine.Error(engine.TypeErrorAtom(temp))
	}
}

// HostPort succeeds if hostPort is an atom consisting of an atom host, a colon, and an integer port.
func HostPort(hostPort, host, port engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	switch hp := env.Resolve(hostPort).(type) {
	case engine.Variable:
		switch h := env.Resolve(host).(type) {
		case engine.Variable:
			return engine.Error(engine.ErrInstantiation)
		case engine.Atom:
			switch p := env.Resolve(port).(type) {
			case engine.Variable:
				return engine.Error(engine.ErrInstantiation)
			case engine.Integer:
				return engine.Unify(hostPort, engine.Atom(net.JoinHostPort(string(h), strconv.Itoa(int(p)))), k, env)
			default:
				return engine.Error(engine.TypeErrorInteger(port))
			}
		default:
			return engine.Error(engine.TypeErrorAtom(host))
		}
	case engine.Atom:
		h, p, err := net.SplitHostPort(string(hp))
		if err != nil {
			return engine.Error(engine.DomainError("host_port", hp))
		}
		po, _ := strconv.Atoi(p)
		given := engine.Compound{Args: []engine.Term{host, port}}
		actual := engine.Compound{Args: []engine.Term{engine.Atom(h), engine.Integer(po)}}
		return engine.Unify(&given, &actual, k, env)
	default:
		return engine.Error(engine.TypeErrorAtom(hostPort))
	}
}

var clientDo = (*http.Client).Do

// Probe probes by making an HTTP request to the target via the proxy.
func Probe(proxy, target, options, status engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	var c http.Client
	switch p := env.Resolve(proxy).(type) {
	case engine.Variable:
		return engine.Error(engine.ErrInstantiation)
	case engine.Atom:
		u, err := ParseURL(string(p))
		if err != nil {
			return engine.Error(engine.DomainError("url", p))
		}

		t := http.DefaultTransport.(*http.Transport).Clone()
		t.Proxy = http.ProxyURL(u)
		c.Transport = t
	default:
		return engine.Error(engine.TypeErrorAtom(proxy))
	}

	var req *http.Request
	switch t := env.Resolve(target).(type) {
	case engine.Variable:
		return engine.Error(engine.ErrInstantiation)
	case engine.Atom:
		var err error
		req, err = http.NewRequest(http.MethodGet, string(t), nil)
		if err != nil {
			return engine.Error(engine.DomainError("url", t))
		}
	default:
		return engine.Error(engine.TypeErrorAtom(t))
	}

	if err := probeOptions(&c, req, options, env); err != nil {
		return engine.Error(err)
	}

	res, err := clientDo(&c, req)
	if err != nil {
		return engine.Bool(false)
	}
	_ = res.Body.Close()

	return engine.Unify(status, engine.Integer(res.StatusCode), k, env)
}

func probeOptions(_ *http.Client, req *http.Request, pairs engine.Term, env *engine.Env) error {
	iter := engine.ListIterator{List: pairs, Env: env}
	for iter.Next() {
		elem := iter.Current()
		switch e := env.Resolve(elem).(type) {
		case engine.Variable:
			return engine.ErrInstantiation
		case *engine.Compound:
			if e.Functor != "-" || len(e.Args) != 2 {
				break
			}

			k, ok := env.Resolve(e.Args[0]).(engine.Atom)
			if !ok {
				return engine.TypeErrorAtom(e.Args[0])
			}

			var vs []string
			iter := engine.ListIterator{List: e.Args[1], Env: env}
			for iter.Next() {
				switch v := env.Resolve(iter.Current()).(type) {
				case engine.Variable:
					return engine.ErrInstantiation
				case engine.Atom:
					vs = append(vs, string(v))
				default:
					return engine.TypeErrorAtom(v)
				}
			}
			if err := iter.Err(); err != nil {
				return err
			}
			req.Header[string(k)] = vs
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}

var logLevels = map[engine.Atom]func(*zerolog.Logger) *zerolog.Event{
	"debug": (*zerolog.Logger).Debug,
	"info":  (*zerolog.Logger).Info,
	"warn":  (*zerolog.Logger).Warn,
	"error": (*zerolog.Logger).Error,
}

func Log(level, msg, pairs engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	return engine.Delay(func(ctx context.Context) *engine.Promise {
		log, ok := ctx.Value(LogKey).(*zerolog.Logger)
		if !ok {
			return engine.Bool(false)
		}

		var e *zerolog.Event
		switch l := env.Resolve(level).(type) {
		case engine.Variable:
			return engine.Error(engine.ErrInstantiation)
		case engine.Atom:
			f, ok := logLevels[l]
			if !ok {
				return engine.Error(engine.DomainError("log_level", level))
			}
			e = f(log)
		default:
			return engine.Error(engine.TypeErrorAtom(level))
		}

		iter := engine.ListIterator{List: pairs, Env: env}
		for iter.Next() {
			elem := iter.Current()
			switch pair := env.Resolve(elem).(type) {
			case engine.Variable:
				return engine.Error(engine.ErrInstantiation)
			case *engine.Compound:
				if pair.Functor != "-" || len(pair.Args) != 2 {
					continue
				}

				switch k := env.Resolve(pair.Args[0]).(type) {
				case engine.Atom:
					switch v := env.Resolve(pair.Args[1]).(type) {
					case engine.Variable:
						return engine.Error(engine.ErrInstantiation)
					case engine.Atom:
						e = e.Str(string(k), string(v))
					case engine.Integer:
						e = e.Int64(string(k), int64(v))
					case engine.Float:
						e = e.Float64(string(k), float64(v))
					}
				}
				continue
			default:
				return engine.Error(engine.TypeErrorPair(elem))
			}
		}
		if err := iter.Err(); err != nil {
			return engine.Error(err)
		}

		switch m := env.Resolve(msg).(type) {
		case engine.Variable:
			return engine.Error(engine.ErrInstantiation)
		case engine.Atom:
			e.Msg(string(m))
			return k(env)
		default:
			return engine.Error(engine.TypeErrorAtom(msg))
		}
	})
}
