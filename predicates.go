package proxima

import (
	"context"
	"github.com/ichiban/prolog/engine"
	"github.com/jtacoma/uritemplates"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"strconv"
)

// URITemplate expands template with Key-Value in pairs list and unifies it with url.
func URITemplate(url, template, pairs engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	switch temp := env.Resolve(template).(type) {
	case engine.Variable:
		return engine.Error(engine.ErrInstantiation)
	case engine.Atom:
		t, err := uritemplates.Parse(string(temp))
		if err != nil {
			return engine.Error(engine.SystemError(err))
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

		ret, err := t.Expand(values)
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}

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
		return engine.Error(engine.TypeErrorAtom(hostPort))
	}
}

func Probe(proxy, target, options engine.Term, k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
	return engine.Delay(func(ctx context.Context) *engine.Promise {
		var req *http.Request
		switch t := env.Resolve(target).(type) {
		case engine.Variable:
			return engine.Error(engine.ErrInstantiation)
		case engine.Atom:
			var err error
			req, err = http.NewRequest(http.MethodGet, string(t), nil)
			if err != nil {
				return engine.Error(engine.SystemError(err))
			}
		default:
			return engine.Error(engine.TypeErrorAtom(t))
		}

		var c http.Client
		switch p := env.Resolve(proxy).(type) {
		case engine.Variable:
			return engine.Error(engine.ErrInstantiation)
		case engine.Atom:
			u, err := ParseURL(string(p))
			if err != nil {
				return engine.Error(engine.SystemError(err))
			}

			t := http.DefaultTransport.(*http.Transport).Clone()
			t.Proxy = http.ProxyURL(u)
			c.Transport = t
		default:
			return engine.Error(engine.TypeErrorAtom(proxy))
		}

		iter := engine.ListIterator{List: options, Env: env}
		for iter.Next() {

		}
		if err := iter.Err(); err != nil {
			return engine.Error(err)
		}

		res, err := c.Do(req)
		if err != nil {
			return engine.Error(engine.SystemError(err))
		}

		if err := res.Body.Close(); err != nil {
			return engine.Error(engine.SystemError(err))
		}

		if res.StatusCode/100 != 2 {
			return engine.Bool(false)
		}

		return k(env)
	})
}

func termHeader(pairs engine.Term, env *engine.Env) (http.Header, error) {
	ret := http.Header{}
	iter := engine.ListIterator{List: pairs, Env: env}
	for iter.Next() {
		elem := iter.Current()
		switch e := env.Resolve(elem).(type) {
		case engine.Variable:
			return nil, engine.ErrInstantiation
		case *engine.Compound:
			if e.Functor != "-" || len(e.Args) != 2 {
				break
			}

			k, ok := env.Resolve(e.Args[0]).(engine.Atom)
			if !ok {
				break
			}

			var vs []string
			iter := engine.ListIterator{List: e.Args[1], Env: env}
			for iter.Next() {
				v, ok := env.Resolve(elem).(engine.Atom)
				if !ok {
					return nil, engine.TypeErrorAtom(elem)
				}
				vs = append(vs, string(v))
			}
			if err := iter.Err(); err != nil {
				return nil, err
			}
			ret[string(k)] = vs
		default:
			break
		}
		return nil, engine.DomainError("header", elem)
	}
	if err := iter.Err(); err != nil {
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

var logLevels = map[engine.Atom]func(*zerolog.Logger) *zerolog.Event{
	"debug": (*zerolog.Logger).Debug,
	"info":  (*zerolog.Logger).Info,
	"warn":  (*zerolog.Logger).Warn,
	"error": (*zerolog.Logger).Error,
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
