package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type Order struct {
	ID      string   `json:"id"`
	Content []string `json:"content"`
}

type OrderDecision struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
}

// WaitSeconds is the number of seconds to wait when the order is accepted
var SleepSeconds = 3

func handleEatsOrderWorkflow(ctx workflow.Context, userId string, order Order, restaurantId string) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("HandleEatsOrder workflow started")

	// Step 1: print that the order was received, along with its details.
	var printReceivedResult string
	err := workflow.ExecuteActivity(ctx, printReceivedActivity, userId, order, restaurantId).Get(ctx, &printReceivedResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	// Step 2: create a channel to receive the order decision
	decisionChan := workflow.GetSignalChannel(ctx, "order-decision")
	var decision OrderDecision

	// Wait for the decision signal
	decisionChan.Receive(ctx, &decision)

	// Step 3: if we're here, the signal was received.
	// Check if the order was accepted or rejected.
	if !decision.Accepted {
		logger.Info("Order rejected", zap.String("reason", decision.Reason))
		// When the order is rejected, we end after logging the reason.
		// We consider this valid execution so it doesn't need an error or retry.
		return nil // Early return
	}

	// If we're here, the order was accepted.
	logger.Info("Order accepted", zap.String("reason", decision.Reason))

	// Step 4: sleep for SleepSeconds seconds (usually 3 seconds)
	workflow.Sleep(ctx, time.Duration(SleepSeconds)*time.Second)
	logger.Info("Waited "+strconv.Itoa(SleepSeconds)+" seconds", zap.Int("seconds", SleepSeconds))

	// Step 5: start the delivery workflow, and wait for it to complete.
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

	err = workflow.ExecuteChildWorkflow(ctx, deliverOrderWorkflow, order.ID).Get(ctx, nil)
	if err != nil {
		logger.Error("Delivery workflow failed", zap.Error(err))
		return err
	}

	// Step 6: print final message after delivery is complete
	var customerMessage string
	err = workflow.ExecuteActivity(ctx, printCustomerMessageActivity).Get(ctx, &customerMessage)
	if err != nil {
		logger.Error("Customer message activity failed", zap.Error(err))
		return err
	}
	logger.Info("Customer message", zap.String("message", customerMessage))

	// End the workflow
	logger.Info("Workflow completed.", zap.String("Result", printReceivedResult))
	return nil
}

func printReceivedActivity(ctx context.Context, userId string, order Order, restaurantId string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("printReceived activity started")
	// Output the content list as a comma-separated string
	contentString := strings.Join(order.Content, ", ")
	return fmt.Sprintf("Order %v received: [%v] from userId %v for restaurantId %v", order.ID, contentString, userId, restaurantId), nil
}

func printCustomerMessageActivity(ctx context.Context) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Printing customer message")
	return "Your order is in front of your door!", nil
}

func init() {
	workflow.Register(handleEatsOrderWorkflow)
	activity.Register(printReceivedActivity)
	activity.Register(printCustomerMessageActivity)
}
