package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
)

type EnqueueURLRequest struct {
	URL string
}

type EnqueueURLResponse struct {
	Status string
}

func HandleRequest(ctx context.Context, event EnqueueURLRequest) (EnqueueURLResponse, error) {
	return EnqueueURLResponse{
		Status: fmt.Sprintf("Enqueued: %s", event.URL),
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
