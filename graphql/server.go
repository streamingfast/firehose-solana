package graphql

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/dfuse-io/logging"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-solana/graphql/apollo"
	"github.com/dfuse-io/dfuse-solana/graphql/resolvers"
	"github.com/dfuse-io/dfuse-solana/graphql/static"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"go.uber.org/zap"
)

type Server struct {
	servingAddr string
	rpcRUL      string
	wsURL       string
}

func NewServer(servingAddr string, rpcRUL string, wsURL string) *Server {
	return &Server{
		servingAddr: servingAddr,
		rpcRUL:      rpcRUL,
		wsURL:       wsURL,
	}
}

func (s *Server) Launch() error {
	// initialize GraphQL
	box := rice.MustFindBox("build")

	resolver := resolvers.NewRoot(s.rpcRUL, s.wsURL)
	schema, err := graphql.ParseSchema(
		string(box.MustBytes("schema.graphql")),
		resolver,
		graphql.UseFieldResolvers(),
		graphql.UseStringDescriptions(),
		graphql.Logger(&graphqlLogger{}),
	)
	derr.Check("parse schema", err)
	return StartHTTPServer(s.servingAddr, schema)

}

func StartHTTPServer(listenAddr string, schema *graphql.Schema) error {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if derr.IsShuttingDown() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	staticRouter := router.PathPrefix("/").Subrouter()
	static.RegisterStaticRoutes(staticRouter)

	restRouter := router.PathPrefix("/").Subrouter()
	restRouter.Use(apollo.NewMiddleware(schema).Handler)
	restRouter.Handle("/graphql", &relay.Handler{Schema: schema})

	// http
	httpListener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		zlog.Panic("http listen failed", zap.String("http_addr", listenAddr), zap.Error(err))
	}

	corsMiddleware := NewCORSMiddleware()
	httpServer := http.Server{
		Handler: corsMiddleware(router),
	}

	zlog.Info("serving HTTP", zap.String("http_addr", listenAddr))
	return httpServer.Serve(httpListener)
}

func NewCORSMiddleware() mux.MiddlewareFunc {
	allowedHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "X-Eos-Push-Guarantee"})
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "OPTIONS"})
	maxAge := handlers.MaxAge(86400) // 24 hours - hard capped by Firefox / Chrome is max 10 minutes

	return handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods, maxAge)
}

type graphqlLogger struct{}

func (*graphqlLogger) LogPanic(ctx context.Context, value interface{}) {
	var err error
	if v, ok := value.(error); ok {
		err = v
	} else {
		err = fmt.Errorf("unknown error: %s", value)
	}
	logging.Logger(ctx, zlog).Error("graphlql resolver panicked", zap.Error(err))
}
