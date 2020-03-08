package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	SQSURL string `envconfig:"SQS_URL" required:"true"`
}

type EnqueueURLRequest struct {
	URL string
}

type EnqueueURLResponse struct {
	Status    string
	MessageID *string
}

type Message struct {
	URL string
}

var config Config
var sqsClient *sqs.SQS

func HandleRequest(ctx context.Context, event EnqueueURLRequest) (EnqueueURLResponse, error) {
	encodedBody, err := json.Marshal(Message{
		URL: event.URL,
	})
	if err != nil {
		return EnqueueURLResponse{}, err
	}

	result, err := sqsClient.SendMessage(&sqs.SendMessageInput{
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Sender": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("getchanski-lambda-0"),
			},
		},
		MessageBody: aws.String(string(encodedBody)),
		QueueUrl:    aws.String(config.SQSURL),
	})

	if err != nil {
		return EnqueueURLResponse{}, err
	}

	log.Println("URL enqueued with id:", result.MessageId)

	return EnqueueURLResponse{
		Status:    "OK",
		MessageID: result.MessageId,
	}, nil
}

func initSQS() error {
	s, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})

	if err != nil {
		return err
	}

	sqsClient = sqs.New(s)
	return nil
}

func main() {
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalln("Bad config:", err.Error())
	}

	if err := initSQS(); err != nil {
		log.Fatalln("Can't create sqs client:", err.Error())
	}

	lambda.Start(HandleRequest)
}
