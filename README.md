# DynamoDB with Go

Series of posts on how to use DynamoDB with Go SDK.

Each episode has it's directory with text of the post and runnable code.

## Table of contents
- [Episode #1 - Setup](./episode1/post.md)
- [Episode #2 - Put & Get](./episode2/post.md)
- [Episode #3 - Build filesystem](./episode3/post.md)

## Prerequisites
1. Golang (1.14 or higher) installed 
2. Docker (19.03 or higher) installed

## Running code 

1. Run local DynamoDB in separate terminal
```
docker run --rm -p 8000:8000 amazon/dynamodb-local
```

2. Execute tests 
```
go test ./...
```