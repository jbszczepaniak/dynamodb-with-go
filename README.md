# dynamodb-with-go

Series of posts on how to use DynamoDB with Go SDK.

Each episode has it's directory with text of the post and runnable code.

## Prerequisites
1. Golang (1.14 or higher) installed 
2. Docker (19.03 or higher) installed

## Running code 

1. Run local DynamoDB in separate terminal
```
docker run -p 8000:8000 amazon/dynamodb-local
```

2. Execute tests 
```
go test
```