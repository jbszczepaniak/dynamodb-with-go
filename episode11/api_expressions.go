package episode12

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func GetItemCollectionV2(ctx context.Context, db dynamodbiface.DynamoDBAPI, table, pk string) ([]Item, error) {
	expr, err := expression.NewBuilder().
		WithKeyCondition(expression.KeyEqual(expression.Key("pk"), expression.Value(pk))).
		Build()

	if err != nil {
		return nil, err
	}
	out, err := db.QueryWithContext(ctx, &dynamodb.QueryInput{
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		TableName:                 aws.String(table),
	})
	if err != nil {
		return nil, err
	}

	var items []Item
	err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func UpdateAWhenBAndUnsetBV2(ctx context.Context, db dynamodbiface.DynamoDBAPI, table string, k Key, newA, whenB string) (Item, error) {
	marshaledKey, err := dynamodbattribute.MarshalMap(k)
	if err != nil {
		return Item{}, err
	}

	expr, err := expression.NewBuilder().
		WithCondition(expression.Equal(expression.Name("b"), expression.Value(whenB))).
		WithUpdate(expression.
			Set(expression.Name("a"), expression.Value(newA)).
			Remove(expression.Name("b"))).
		Build()
	if err != nil {
		return Item{}, err
	}
	out, err := db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		Key:                       marshaledKey,
		ReturnValues:              aws.String("ALL_NEW"),
		TableName:                 aws.String(table),
	})
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return Item{}, fmt.Errorf("b is not %s, aborting update", whenB)
		}
		return Item{}, err
	}
	var i Item
	err = dynamodbattribute.UnmarshalMap(out.Attributes, &i)
	if err != nil {
		return Item{}, err
	}
	return i, nil
}

func PutIfNotExistsV2(ctx context.Context, db dynamodbiface.DynamoDBAPI, table string, k Key) error {
	marshaledKey, err := dynamodbattribute.MarshalMap(k)
	if err != nil {
		return err
	}

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeNotExists(expression.Name("pk"))).
		Build()
	if err != nil {
		return err
	}

	_, err = db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		ConditionExpression:      expr.Condition(),
		ExpressionAttributeNames: expr.Names(),
		Item:                     marshaledKey,
		TableName:                aws.String(table),
	})

	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return errors.New("Item with this Key already exists")
		}
		return err
	}

	return nil
}

