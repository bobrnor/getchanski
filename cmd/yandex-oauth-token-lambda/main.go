package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	uuid "github.com/satori/go.uuid"
)

type Request struct {
	AccessToken      string
	ExpiresIn        int
	TokenType        string
	Error            string
	ErrorDescription string
}

type Response struct {
	Status string
	UUID   string
	Cookie string
}

type YDUserInfo struct {
	Country string `json:"country"`
	Login   string `json:"login"`
	UID     string `json:"uid"`
}

type YDError struct {
	Message     string `json:"message"`
	Description string `json:"description"`
	Error       string `json:"error"`
}

type YDDiskGetResponse struct {
	YDError
	User YDUserInfo `json:"user"`
}

type UserItem struct {
	UUID      string    `json:"uuid"`
	UserID    string    `json:"user_id"`
	Login     string    `json:"login"`
	Country   string    `json:"country"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func getUserInfo(token string) (YDUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, "https://cloud-api.yandex.net:443/v1/disk", nil)
	if err != nil {
		return YDUserInfo{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return YDUserInfo{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return YDUserInfo{}, err
	}

	var decodedBody YDDiskGetResponse
	if err := json.Unmarshal(body, &decodedBody); err != nil {
		return YDUserInfo{}, fmt.Errorf("can't unmarshal %s: %w", body, err)
	}

	if len(decodedBody.Error) > 0 {
		return YDUserInfo{}, fmt.Errorf("%s: %s (%s)", decodedBody.Error, decodedBody.Message, decodedBody.Description)
	}

	return decodedBody.User, nil
}

var dynamoDBClient *dynamodb.DynamoDB

func getDynamoDBClient() (*dynamodb.DynamoDB, error) {
	if dynamoDBClient != nil {
		return dynamoDBClient, nil
	}

	s, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})

	if err != nil {
		return nil, err
	}

	return dynamodb.New(s), nil
}

func HandleRequest(ctx context.Context, event Request) (Response, error) {
	if len(event.Error) > 0 {
		log.Println(event.Error, event.ErrorDescription)
		return Response{
			Status: "ErrOK",
			UUID:   "",
		}, nil
	}

	userInfo, err := getUserInfo(event.AccessToken)
	if err != nil {
		return Response{}, err
	}

	userUUID := uuid.NewV4()

	userItem := UserItem{
		UUID:      userUUID.String(),
		UserID:    userInfo.UID,
		Login:     userInfo.Login,
		Country:   userInfo.Country,
		Token:     event.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(event.ExpiresIn) * time.Second),
	}

	ddbItem, err := dynamodbattribute.MarshalMap(userItem)
	if err != nil {
		return Response{}, err
	}

	input := &dynamodb.PutItemInput{
		Item:      ddbItem,
		TableName: aws.String("getchanski-users"),
	}

	db, err := getDynamoDBClient()
	if err != nil {
		return Response{}, err
	}

	_, err = db.PutItem(input)
	if err != nil {
		return Response{}, err
	}

	log.Println(userInfo)

	return Response{
		Status: "OK",
		UUID:   userItem.UUID,
		Cookie: fmt.Sprintf("uuid=%s; domain=getchanski-site.s3-website-us-east-1.amazonaws.com; expires=%s; HttpOnly;", userItem.UUID, userItem.ExpiresAt.Format(http.TimeFormat)),
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
