package episode3

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

func TestSingleFileFromDirectory(t *testing.T) {
	ctx := context.Background()
	tableName := "FileSystemTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	insert(ctx, db, tableName)

	out, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"directory": &types.AttributeValueMemberS{Value: "finances"},
			"filename":  &types.AttributeValueMemberS{Value: "report2020.pdf"},
		},
		TableName: aws.String(tableName),
	})
	assert.NoError(t, err)

	var i item
	err = attributevalue.UnmarshalMap(out.Item, &i)
	assert.NoError(t, err)
	assert.Equal(t, item{Directory: "finances", Filename: "report2020.pdf", Size: "2MB"}, i)
}

func TestAllFilesFromDirectory(t *testing.T) {
	ctx := context.Background()
	tableName := "FileSystemTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	insert(ctx, db, tableName)

	expr, err := expression.NewBuilder().
		WithKeyCondition(
			expression.KeyEqual(expression.Key("directory"), expression.Value("finances"))).
		Build()
	assert.NoError(t, err)

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		TableName:                 aws.String(tableName),
	})
	assert.NoError(t, err)
	assert.Len(t, out.Items, 4)
}

func TestAllReportsBefore2019(t *testing.T) {
	ctx := context.Background()
	tableName := "FileSystemTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	insert(ctx, db, tableName)

	expr, err := expression.NewBuilder().
		WithKeyCondition(
			expression.KeyAnd(
				expression.KeyEqual(expression.Key("directory"), expression.Value("finances")),
				expression.KeyLessThan(expression.Key("filename"), expression.Value("report2019")))).
		Build()
	assert.NoError(t, err)

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		TableName:                 aws.String(tableName),
	})
	assert.NoError(t, err)
	var items []item
	err = attributevalue.UnmarshalListOfMaps(out.Items, &items)
	assert.NoError(t, err)
	if assert.Len(t, items, 2) {
		assert.Equal(t, "report2017.pdf", items[0].Filename)
		assert.Equal(t, "report2018.pdf", items[1].Filename)
	}
}

type item struct {
	Directory string `dynamodbav:"directory"`
	Filename  string `dynamodbav:"filename"`
	Size      string `dynamodbav:"size"`
}

func insert(ctx context.Context, db *dynamodb.Client, tableName string) {
	item1 := item{Directory: "finances", Filename: "report2017.pdf", Size: "1MB"}
	item2 := item{Directory: "finances", Filename: "report2018.pdf", Size: "1MB"}
	item3 := item{Directory: "finances", Filename: "report2019.pdf", Size: "1MB"}
	item4 := item{Directory: "finances", Filename: "report2020.pdf", Size: "2MB"}
	item5 := item{Directory: "fun", Filename: "game1", Size: "4GB"}

	for _, item := range []item{item1, item2, item3, item4, item5} {
		attrs, _ := attributevalue.MarshalMap(&item)
		db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      attrs,
		})
	}
}
