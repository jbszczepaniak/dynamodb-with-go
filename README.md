# DynamoDB with Go

:warning:
DynamoDB with Go was updated to use new (v2) version of AWS SDK for Go. If you want to play with older version
just checkout branch `go-sdk-v1`.
:warning:

Series of posts on how to use DynamoDB with Go SDK.

Each episode has it's directory with text of the post and runnable code.

## Table of contents
- [Episode #1 - Setup](./episode1/post.md)
- [Episode #2 - Put & Get](./episode2/post.md)
- [Episode #3 - Composite Primary Keys](./episode3/post.md)
- [Episode #4 - Indices ](./episode4/post.md)
- [Episode #5 - Legacy IDs mapping](./episode5/post.md)
- [Episode #6 - Legacy IDs mapping with transactions](./episode6/post.md)
- [Episode #7 - Modelling hierarchical data with Single Table Design](./episode7/post.md)
- [Episode #8 - Implement hierarchical data with Single Table Design](./episode8/post.md)
- [Episode #9 - Switching the toggle, toggling the switch](./episode9/post.md)
- [Episode #10 - Gotcha with empty slices](./episode10/post.md)
- [Episode #11 - Expressions API](./episode11/post.md)
- [Episode #12 - Condition on other item from item collection](./episode12/post.md)

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