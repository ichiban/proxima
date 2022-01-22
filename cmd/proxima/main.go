package main

import (
	"context"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"os/signal"
	"proxima"
	"time"

	"github.com/justinas/alice"
)

func main() {
	flag.Parse()

	w := io.Writer(os.Stderr)
	if term.IsTerminal(2) { // stderr
		w = zerolog.ConsoleWriter{Out: os.Stderr}
	}

	log := zerolog.New(w).With().
		Timestamp().
		Logger()

	s, err := proxima.Open(flag.Args())
	if err != nil {
		log.Fatal().Err(err).Msg("proxima.Open() failed")
	}

	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt)
	defer close()

	serve(ctx, s, log)
}

func serve(ctx context.Context, s *proxima.Switcher, log zerolog.Logger) {
	var result struct {
		Addr string
	}
	if err := s.QuerySolutionContext(ctx, `listen(Addr).`).Scan(&result); err != nil {
		log.Fatal().Err(err).Msg("listen(Addr) failed")
	}

	srv := http.Server{
		Addr:    result.Addr,
		Handler: handler(s, log),
	}
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("srv.Shutdown() failed")
		}
	}()
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("srv.ListenAndServe() failed")
	}
}

func handler(s *proxima.Switcher, log zerolog.Logger) http.Handler {
	return alice.New(
		hlog.NewHandler(log),
		hlog.RemoteAddrHandler("remote"),
		hlog.UserAgentHandler("ua"),
		hlog.RequestIDHandler("rid", "Request-Id"),
	).Then(s)
}
