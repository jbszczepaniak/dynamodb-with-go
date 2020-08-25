package episode2

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/stretchr/testify/assert"
)

type Order struct {
	ID        string `dynamodbav:"id"`
	Price     int    `dynamodbav:"price"`
	IsShipped bool   `dynamodbav:"is_shipped"`
}

func TestPutGet(t *testing.T) {
	ctx := context.Background()
	tableName := "OrdersTable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	order := Order{ID: "12-34", Price: 22, IsShipped: false}
	avs, err := dynamodbattribute.MarshalMap(order)
	assert.NoError(t, err)

	_, err = db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      avs,
	})
	assert.NoError(t, err)

	out, err := db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String("12-34"),
			},
		},
		TableName: aws.String(tableName),
	})
	assert.NoError(t, err)

	var queried Order
	err = dynamodbattribute.UnmarshalMap(out.Item, &queried)
	assert.NoError(t, err)
	assert.Equal(t, Order{ID: "12-34", Price: 22, IsShipped: false}, queried)

}
