package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/kelseyhightower/envconfig"
	"github.com/sanity-io/litter"
)

var config Config
var sqsClient *sqs.SQS

type Config struct {
	SQSURL                  string `envconfig:"SQS_URL" required:"true"`
	SQSLongpollTimeoutInSec int64  `envconfig:"SQS_LONGPOLL_TIMEOUT_IN_SEC" default:"10"`
}

type Message struct {
	URL string
}

func loop() error {
	for {
		result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			AttributeNames: aws.StringSlice([]string{
				"All",
			}),
			MaxNumberOfMessages: aws.Int64(1),
			MessageAttributeNames: aws.StringSlice([]string{
				"All",
			}),
			QueueUrl:        aws.String(config.SQSURL),
			WaitTimeSeconds: aws.Int64(config.SQSLongpollTimeoutInSec),
		})

		if err != nil {
			log.Println("Can't receive message from sqs:", err.Error())
			time.Sleep(30 * time.Second)
			continue
		}

		for _, m := range result.Messages {
			if m == nil || m.Body == nil {
				log.Println("Empty message:", litter.Sdump(m))
				continue
			}

			var decodedBody Message
			if err := json.Unmarshal([]byte(*m.Body), &decodedBody); err != nil {
				log.Println("Can't unmarshal message:", litter.Sdump(m), ",", err.Error())
				continue
			}

			// TODO: process url
			log.Println("Proceccing URL:", decodedBody.URL)

			result, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(config.SQSURL),
				ReceiptHandle: m.ReceiptHandle,
			})

			if err != nil {
				log.Println("Can't delete message from sqs:", litter.Sdump(m), ",", err.Error())
				continue
			}

			log.Println("Message deleted:", *m.MessageId, ",", result)
		}
	}
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

	if err := loop(); err != nil {
		log.Fatalln("Error during loop:", err.Error())
	}
}
