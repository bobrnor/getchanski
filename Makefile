VERSION:=$(shell git log --date=short --pretty=format:'%ad-%h' -n 1)

build-enqueue:
	GOOS=linux go build -o enqueue cmd/enqueue-lambda/main.go
	zip function.zip enqueue

build-n-push-worker:
	aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 783868845322.dkr.ecr.us-east-1.amazonaws.com/getchanski
	GOOS=linux go build -o deploy/worker/worker cmd/worker/main.go
	docker build --rm=true -t getchanski:$(VERSION) -t783868845322.dkr.ecr.us-east-1.amazonaws.com/getchanski:$(VERSION) deploy/worker/.
	docker push 783868845322.dkr.ecr.us-east-1.amazonaws.com/getchanski:$(VERSION)