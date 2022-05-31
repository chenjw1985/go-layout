package main

import (
	"flag"
	"os"

	"github.com/davidchen-cn/go-layout/internal/conf"
	"github.com/go-kratos/kratos/contrib/config/apollo/v2"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	clientV3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

const (
	KratosOnDev = "dev"
)

var (
	AppName    = "go-layout"
	AppVersion = "0.1.0"
	id, _      = os.Hostname()
	// flagConf is the config flag.
	flagConf string
)

func init() {
	flag.StringVar(&flagConf, "conf", "../../configs/config.yaml", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, hs *http.Server, gs *grpc.Server, etcdConf *conf.Etcd) *kratos.App {
	opts := []kratos.Option{
		kratos.ID(id),
		kratos.Name(AppName),
		kratos.Version(AppVersion),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			gs,
		),
	}
	// enable etcd as services discoverer
	if etcdConf != nil && etcdConf.Hosts != nil {
		log.Debugf("Init etcd services discoverer on %v", etcdConf.Hosts)
		client, err := clientV3.New(clientV3.Config{
			Endpoints: etcdConf.Hosts,
		})
		if err != nil {
			panic(err)
		}
		// new reg with etcd client
		reg := etcd.New(client)
		opts = append(opts, kratos.Registrar(reg))
	}

	return kratos.New(opts...)
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

func createTracerProvider(endpoint string) (*tracesdk.TracerProvider, error) {
	environment := "prod"
	if os.Getenv("KratosRunMode") != "" {
		environment = os.Getenv("KratosRunMode")
	}
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Set the sampling rate based on the parent span to 100%
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(1.0))),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(AppName),
			attribute.String("environment", environment),
			attribute.String("ID", id),
		)),
	)
	return tp, nil
}

func main() {
	flag.Parse()
	// init config
	apolloConf := getApolloConfig()
	var c config.Config
	if mode := os.Getenv("KratosRunMode"); mode == KratosOnDev {
		log.Debugf("Init config from %s", flagConf)
		c = config.New(
			config.WithSource(
				file.NewSource(flagConf),
			),
		)
	} else {
		log.Debugf("Init config from %s:%s(%s)", apolloConf.AppID, apolloConf.Endpoint, apolloConf.Namespace)
		c = config.New(
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
	}
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// init tracing
	if bc.Application.Trace != nil && bc.Application.Trace.Endpoint != "" {
		log.Debugf("Init tracing services on %s", bc.Application.Trace.Endpoint)
		traceTp, err := createTracerProvider(bc.Application.Trace.Endpoint)
		if err != nil {
			panic(err)
		}
		otel.SetTracerProvider(traceTp)
	}

	// init logger
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", AppName,
		"service.version", AppVersion,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	// Bootstrap
	app, cleanup, err := wireApp(bc.Application.Server, bc.Application.Data, bc.Application.Etcd, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
