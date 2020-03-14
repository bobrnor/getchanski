package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

type Request struct{}

type Response struct {
	RedirectURL string
}

func HandleRequest(ctx context.Context, event Request) (Response, error) {
	return Response{
		RedirectURL: "https://oauth.yandex.ru/authorize?response_type=token&client_id=6254bbf47aa3496ba815f448f7f720de",
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
