# DynamoDB with Go #1 - Setup

This post begins short series that aims to explore Go API used when interacting with DynamoDB database.
Such combination of Go and DynamoDB is being used in serverless applications - specifically ones that run
on the GoLD stack, where Go is the runtime for the AWS Lambda and DynamoDB is a database of choice.

Make sure to read [Introduction to the GoLD stack](https://dev.to/prozz/introduction-to-the-gold-stack-5b66)
by [@prozz](https://twitter.com/prozz).

Throughout the series we are going to learn how to use the API in a convenient way. We are going to show
some popular use cases, we are going to learn tips, tricks, and we are going to fight gotchas
of that API.

## [Setting up the stage](#setting-up-stage)

The goal of this very first post in the series is to setup environment for us. At the end of
this post I would like to run simple API call that returns connection to the DynamoDB that I
can play with. Before that happens we need to have the following dependencies.

1. Golang 1.x (I am using 1.15) installed 
2. Docker 19.03 (or higher) installed

Next step is to clone the repository.
```
git clone git@github.com:jbszczepaniak/dynamodb-with-go.git
```

That's it. Now we can run local DynamoDB.

```
docker run --rm -p 8000:8000 amazon/dynamodb-local
```

This will take entire terminal session. The advantage is that after you are done playing
with DynamoDB - you will remember to shut it down. If you want to run DynamoDB in the background
add `-d` parameter to the `docker run` command. Either way - since you are running the local
version of DynamoDB you can go to the directory where repository was cloned and run.

```
go test ./... -v
```

The idea for this series is for you to always be able to run container with DynamoDB and execute
test suite. Having working examples and being able to play with them is excellent opportunity to learn!

## [Creating DynamoDB tables](#creating-tables)

This series will be driven by tests. We are going to setup DynamoDB table, act on it in some ways
and verify what happened. We already have environment up and running. Now we need to have code that will
create the DynamoDB tables for us. I created dynamo `package` that provides `SetupTable` test helper
that takes path to the CloudFormation template file and table name and creates that table for us.

Let me show you test that demonstrates usage of the [`SetupTable`](../pkg/dynamo/setup_test.go).

```go
ctx := context.Background()
db, cleanup := SetupTable(t, ctx, "PartitionKeyTable", "./testdata/template.yml")
``` 

`PartitionKeyTable` is the name of the DynamoDB table that is defined in the [`template.yml`](../pkg/dynamo/testdata/template.yml)
file. File itself follows format of CloudFormation templates.

`SetupTable` returns `db` - connection to the database, and `cleanup` method which needs to be called
after every test. You cannot have many tables with the same name in DynamoDB - we need to clean them up.

```go
out, err := db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
  TableName: aws.String("PartitionKeyTable"),
})
assert.NoError(t, err)
assert.Equal(t, "PartitionKeyTable", *out.Table.TableName)
assert.Equal(t, "pk", *out.Table.AttributeDefinitions[0].AttributeName)
```

Next piece of the test asks DynamoDB about the table we've just created. Upon receiving the answer - we check
whether table name and name of the Partition Key matches specification from CloudFormation template. We will
talk about different types of keys in following episodes of the series.

```go
cleanup()
_, err = db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
    TableName: aws.String("PartitionKeyTable"),
})

var notfound *types.ResourceNotFoundException
assert.True(t, errors.As(err, &notfound))
```

At the end of the test we run the `cleanup()` and verify that DynamoDB doesn't anything about it anymore.

## [Summary](#summary)

We've just prepared ourselves for the journey of exploring Go API for DynamoDB. We can create DynamoDB tables
out of CloudFormation templates in our local instance of DynamoDB that runs inside Docker. We can now run tests against
the local instance demonstrating various aspects of the API.
