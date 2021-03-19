package episode4

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestPhotosYoungerThan(t *testing.T) {
	ctx := context.Background()
	tableName := "FileSystemTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	insert(ctx, db, tableName)

	expr, err := expression.NewBuilder().
		WithKeyCondition(
			expression.KeyAnd(
				expression.KeyEqual(expression.Key("directory"), expression.Value("photos")),
				expression.KeyGreaterThanEqual(
					expression.Key("created_at"),
					expression.Value(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC))))).
		Build()
	assert.NoError(t, err)

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		TableName:                 aws.String(tableName),
		IndexName:                 aws.String("ByCreatedAt"),
	})
	assert.NoError(t, err)
	assert.Len(t, out.Items, 2)
}

func TestPhotosFromTimeRange(t *testing.T) {
	ctx := context.Background()
	tableName := "FileSystemTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	insert(ctx, db, tableName)

	expr, err := expression.NewBuilder().
		WithKeyCondition(
			expression.KeyAnd(
				expression.KeyEqual(expression.Key("directory"), expression.Value("photos")),
				expression.KeyBetween(expression.Key("created_at"),
					expression.Value(time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)),
					expression.Value(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC))))).
		Build()
	assert.NoError(t, err)

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		TableName:                 aws.String(tableName),
		IndexName:                 aws.String("ByCreatedAt"),
	})
	assert.NoError(t, err)
	assert.Len(t, out.Items, 2)
}

func TestNewestPhoto(t *testing.T) {
	ctx := context.Background()
	tableName := "FileSystemTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	insert(ctx, db, tableName)

	expr, err := expression.NewBuilder().
		WithKeyCondition(expression.KeyEqual(expression.Key("directory"), expression.Value("photos"))).
		Build()
	assert.NoError(t, err)

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		TableName:                 aws.String(tableName),
		IndexName:                 aws.String("ByCreatedAt"),
		ScanIndexForward:          aws.Bool(false),
		Limit:                     aws.Int32(1),
	})
	assert.NoError(t, err)

	var items []item
	err = attributevalue.UnmarshalListOfMaps(out.Items, &items)
	assert.NoError(t, err)

	assert.Equal(t, 2020, items[0].CreatedAt.Year())
}

type item struct {
	Directory string    `dynamodbav:"directory"`
	Filename  string    `dynamodbav:"filename"`
	Size      string    `dynamodbav:"size"`
	CreatedAt time.Time `dynamodbav:"created_at"`
}

func insert(ctx context.Context, db *dynamodb.Client, tableName string) {
	item1 := item{Directory: "photos", Filename: "bike.png", Size: "1.2MB", CreatedAt: time.Date(2017, 3, 4, 0, 0, 0, 0, time.UTC)}
	item2 := item{Directory: "photos", Filename: "apartment.jpg", Size: "4MB", CreatedAt: time.Date(2018, 6, 25, 0, 0, 0, 0, time.UTC)}
	item3 := item{Directory: "photos", Filename: "grandpa.png", Size: "3MB", CreatedAt: time.Date(2019, 4, 1, 0, 0, 0, 0, time.UTC)}
	item4 := item{Directory: "photos", Filename: "kids.png", Size: "3MB", CreatedAt: time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC)}

	for _, item := range []item{item1, item2, item3, item4} {
		attrs, _ := attributevalue.MarshalMap(&item)
		db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      attrs,
		})
	}
}
