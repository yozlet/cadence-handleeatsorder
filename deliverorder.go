package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// DeliverySleepSeconds is the number of seconds to wait in the delivery workflow
var DeliverySleepSeconds = 4

func deliverOrderWorkflow(ctx workflow.Context, orderID string) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("DeliverOrder workflow started")

	// Step 1: sleep for DeliverySleepSeconds seconds
	workflow.Sleep(ctx, time.Duration(DeliverySleepSeconds)*time.Second)
	logger.Info("Waited "+strconv.Itoa(DeliverySleepSeconds)+" seconds", zap.Int("seconds", DeliverySleepSeconds))

	// Step 2: deliver the order
	var deliveryResult string
	err := workflow.ExecuteActivity(ctx, deliverOrderActivity, orderID).Get(ctx, &deliveryResult)
	if err != nil {
		logger.Error("Delivery activity failed.", zap.Error(err))
		return err
	}

	// End the workflow
	logger.Info("Delivery completed", zap.String("Result", deliveryResult))
	return nil
}

func deliverOrderActivity(ctx context.Context, orderID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Delivering order", zap.String("OrderID", orderID))
	return fmt.Sprintf("Order %s delivered!", orderID), nil
}

func init() {
	workflow.Register(deliverOrderWorkflow)
	activity.Register(deliverOrderActivity)
}
