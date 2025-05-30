package main

import (
    "net/http"
    "go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
    "go.uber.org/cadence/compatibility"
    "go.uber.org/cadence/worker"

    apiv1 "github.com/uber/cadence-idl/go/proto/api/v1"
    // "github.com/uber-go/tally"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "go.uber.org/yarpc"
    "go.uber.org/yarpc/transport/grpc"
)

var HostPort = "127.0.0.1:7833"
var Domain = "eats"
var TaskListName = "eats-worker"
var ClientName = "eats-worker"
var CadenceService = "cadence-frontend"

func main() {
    startWorker(buildLogger(), buildCadenceClient())
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}

func buildLogger() *zap.Logger {
    config := zap.NewDevelopmentConfig()
    config.Level.SetLevel(zapcore.InfoLevel)

    var err error
    logger, err := config.Build()
    if err != nil {
        panic("Failed to setup logger")
    }

    return logger
}

func buildCadenceClient() workflowserviceclient.Interface {
    dispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name: ClientName,
			Outbounds: yarpc.Outbounds{
				CadenceService: {Unary: grpc.NewTransport().NewSingleOutbound(HostPort)},
			},
		})
		if err := dispatcher.Start(); err != nil {
			panic("Failed to start dispatcher")
		}

		clientConfig := dispatcher.ClientConfig(CadenceService)

		return compatibility.NewThrift2ProtoAdapter(
			apiv1.NewDomainAPIYARPCClient(clientConfig),
			apiv1.NewWorkflowAPIYARPCClient(clientConfig),
			apiv1.NewWorkerAPIYARPCClient(clientConfig),
			apiv1.NewVisibilityAPIYARPCClient(clientConfig),
		)
}

func startWorker(logger *zap.Logger, service workflowserviceclient.Interface) {
    // TaskListName identifies set of client workflows, activities, and workers.
    // It could be your group or client or application name.
    workerOptions := worker.Options{
        Logger:       logger,
    //    MetricsScope: tally.NewTestScope(TaskListName, map[string]string{}),
    }

    worker := worker.New(
        service,
        Domain,
        TaskListName,
        workerOptions)
    err := worker.Start()
    if err != nil {
        panic("Failed to start worker")
    }

    logger.Info("Started Worker.", zap.String("worker", TaskListName))
}