package episode13

import (
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/stretchr/testify/assert"
)

func TestInsertingOrderFailsBecauseUserDoesNotExist(t *testing.T) {
	ctx := context.Background()
	tableName := "ATable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeExists(expression.Name("pk"))).
		WithUpdate(expression.Add(expression.Name("orders_count"), expression.Value(1))).
		Build()
	assert.NoError(t, err)

	_, err = db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Put: &dynamodb.Put{
					Item: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String("1234")},
						"sk": {S: aws.String("ORDER#2017-03-04 00:00:00 +0000 UTC")},
					},
					TableName: aws.String(tableName),
				},
			},
			{
				Update: &dynamodb.Update{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeValues: expr.Values(),
					ExpressionAttributeNames:  expr.Names(),
					UpdateExpression:          expr.Update(),
					Key: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String("1234")},
						"sk": {S: aws.String("USERINFO")},
					},
					TableName: aws.String(tableName),
				},
			},
		},
	})

	assert.Error(t, err)
	_, ok := err.(*dynamodb.TransactionCanceledException)
	assert.True(t, ok)
}

func TestInsertingOrderSucceedsBecauseUserExists(t *testing.T) {
	ctx := context.Background()
	tableName := "ATable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	_, err := db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"pk": {S:aws.String("1234")},
			"sk": {S:aws.String("USERINFO")},
		},
		TableName: aws.String(tableName),
	})
	assert.NoError(t ,err)

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeExists(expression.Name("pk"))).
		WithUpdate(expression.Add(expression.Name("orders_count"), expression.Value(1))).
		Build()
	assert.NoError(t, err)

	_, err = db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Put: &dynamodb.Put{
					Item: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String("1234")},
						"sk": {S: aws.String("ORDER#2017-03-04 00:00:00 +0000 UTC")},
					},
					TableName: aws.String(tableName),
				},
			},
			{
				Update: &dynamodb.Update{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeValues: expr.Values(),
					ExpressionAttributeNames:  expr.Names(),
					UpdateExpression:          expr.Update(),
					Key: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String("1234")},
						"sk": {S: aws.String("USERINFO")},
					},
					TableName: aws.String(tableName),
				},
			},
		},
	})
	assert.NoError(t, err)
}
