package episode13

import (
	"dynamodb-with-go/pkg/dynamo"
	"errors"
	"testing"

	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

	_, err = db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					Item: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: "1234"},
						"sk": &types.AttributeValueMemberS{Value: "ORDER#2017-03-04 00:00:00 +0000 UTC"},
					},
					TableName: aws.String(tableName),
				},
			},
			{
				Update: &types.Update{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeValues: expr.Values(),
					ExpressionAttributeNames:  expr.Names(),
					UpdateExpression:          expr.Update(),
					Key: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: "1234"},
						"sk": &types.AttributeValueMemberS{Value: "USERINFO"},
					},
					TableName: aws.String(tableName),
				},
			},
		},
	})

	assert.Error(t, err)
	var transactionCancelled *types.TransactionCanceledException
	assert.True(t, errors.As(err, &transactionCancelled))
}

func TestInsertingOrderSucceedsBecauseUserExists(t *testing.T) {
	ctx := context.Background()
	tableName := "ATable"
	db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
	defer cleanup()

	_, err := db.PutItem(ctx, &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: "1234"},
			"sk": &types.AttributeValueMemberS{Value: "USERINFO"},
		},
		TableName: aws.String(tableName),
	})
	assert.NoError(t, err)

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeExists(expression.Name("pk"))).
		WithUpdate(expression.Add(expression.Name("orders_count"), expression.Value(1))).
		Build()
	assert.NoError(t, err)

	_, err = db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					Item: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: "1234"},
						"sk": &types.AttributeValueMemberS{Value: "ORDER#2017-03-04 00:00:00 +0000 UTC"},
					},
					TableName: aws.String(tableName),
				},
			},
			{
				Update: &types.Update{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeValues: expr.Values(),
					ExpressionAttributeNames:  expr.Names(),
					UpdateExpression:          expr.Update(),
					Key: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: "1234"},
						"sk": &types.AttributeValueMemberS{Value: "USERINFO"},
					},
					TableName: aws.String(tableName),
				},
			},
		},
	})
	assert.NoError(t, err)
}
