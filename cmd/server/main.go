package main

import (
	"flag"
	"os"

	"github.com/davidchen-cn/go-layout/internal/conf"
	"github.com/go-kratos/kratos/contrib/config/apollo/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string

	id, _ = os.Hostname()
)

func newApp(logger log.Logger, hs *http.Server, gs *grpc.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			gs,
		),
	)
}

type ApolloConf struct {
	AppID     string
	Cluster   string
	Endpoint  string
	Namespace string
	Secret    string
}

func getApolloConfig() *ApolloConf {
	conf := &ApolloConf{
		AppID:     os.Getenv("APOLLO_APPID"),
		Cluster:   "default",
		Endpoint:  os.Getenv("APOLLO_ENDPOINT"),
		Namespace: "application.yaml",
		Secret:    os.Getenv("APOLLO_SECRET"),
	}
	if cluster := os.Getenv("APOLLO_CLUSTER"); cluster != "" {
		conf.Cluster = cluster
	}
	if namespace := os.Getenv("APOLLO_NAMESPACE"); namespace != "" {
		conf.Namespace = namespace
	}
	return conf
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	apolloConf := getApolloConfig()
	c := config.New(
		config.WithSource(
			apollo.NewSource(
				apollo.WithAppID(apolloConf.AppID),
				apollo.WithCluster(apolloConf.Cluster),
				apollo.WithEndpoint(apolloConf.Endpoint),
				apollo.WithNamespace(apolloConf.Namespace),
				apollo.WithEnableBackup(),
				apollo.WithSecret(apolloConf.Secret),
			),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
