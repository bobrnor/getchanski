package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/kelseyhightower/envconfig"
	"github.com/sanity-io/litter"
)

var config Config
var sqsClient *sqs.SQS
var s3Client *s3manager.Uploader

type Config struct {
	SQSURL                  string `envconfig:"SQS_URL" required:"true"`
	SQSLongpollTimeoutInSec int64  `envconfig:"SQS_LONGPOLL_TIMEOUT_IN_SEC" default:"10"`
}

type Message struct {
	URL string
}

type MediaInfo struct {
	ID        string `json:"id"`
	FullTitle string `json:"fulltitle"`
	Ext       string `json:"ext"`
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

			log.Println("Proceccing URL, getting info:", decodedBody.URL)
			cmd := exec.Command("youtube-dl",
				"--no-mark-watched",
				"--no-color",
				"-no-playlist",
				"--retries", "10",
				"--dump-json",
				"--no-cache-dir",
				"--no-progress",
				"--no-warnings",
				decodedBody.URL,
			)

			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			if err := cmd.Run(); err != nil {
				log.Println("Can't execute command with youtube-dl:", cmd.String(), ",", out.String(), ",", err.Error())
				continue
			}

			var mediaInfo MediaInfo
			if err := json.Unmarshal(out.Bytes(), &mediaInfo); err != nil {
				log.Println("Can't unmarshal media info:", out.String(), err.Error())
				continue
			}

			log.Println("Proceccing URL, downloading context:", decodedBody.URL)
			cmd = exec.Command("youtube-dl",
				"--no-mark-watched",
				"--no-color",
				"-no-playlist",
				"--retries", "10",
				"--output", "%(id)s.%(ext)s",
				"--no-cache-dir",
				"--no-progress",
				"--format", "bestaudio[ext=mp3]/bestaudio/best",
				"--extract-audio",
				"--audio-format", "mp3",
				decodedBody.URL,
			)

			out = bytes.Buffer{}
			cmd.Stdout = &out
			cmd.Stderr = &out

			if err := cmd.Run(); err != nil {
				log.Println("Can't execute command with youtube-dl:", cmd.String(), ",", out.String(), ",", err.Error())
				continue
			}

			log.Println("youtube-dl:", out.String())

			if err := moveToS3(mediaInfo); err != nil {
				log.Println("Can't move to s3:", err.Error())
				continue
			}

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

func moveToS3(m MediaInfo) error {
	fileName := fmt.Sprintf("%s.mp3", m.ID)
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("Can't open file with media: %s, %w", fileName, err)
	}
	defer file.Close()

	_, err =
		s3Client.Upload(&s3manager.UploadInput{
			Bucket: aws.String("getchanski-storage"),
			Key:    aws.String(fmt.Sprintf("%s.mp3", m.FullTitle)),
			Body:   file,
		})
	if err != nil {
		return fmt.Errorf("Can't upload file to s3: %s, %w", m, err)
	}

	log.Println("File uploaded to S3")

	if err := os.Remove(fileName); err != nil {
		log.Println("Can't remove *.mp3 file:", fileName, err.Error())
	}

	return nil
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

func initS3() error {
	s, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})

	if err != nil {
		return err
	}

	s3Client = s3manager.NewUploader(s)
	return nil
}

func main() {
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalln("Bad config:", err.Error())
	}

	if err := initSQS(); err != nil {
		log.Fatalln("Can't create sqs client:", err.Error())
	}

	if err := initS3(); err != nil {
		log.Fatalln("Can't create s3 client:", err.Error())
	}

	if err := loop(); err != nil {
		log.Fatalln("Error during loop:", err.Error())
	}
}
