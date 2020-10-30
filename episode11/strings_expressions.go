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
)

func GetItemCollectionV1(ctx context.Context, db dynamodbiface.DynamoDBAPI, table, pk string) ([]Item, error) {
	out, err := db.QueryWithContext(ctx, &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("#key = :value"),
		ExpressionAttributeNames: map[string]*string{
			"#key": aws.String("pk"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":value": {S: aws.String(pk)},
		},
		TableName: aws.String(table),
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

func UpdateAWhenBAndUnsetBV1(ctx context.Context, db dynamodbiface.DynamoDBAPI, table string, k Key, newA, whenB string) (Item, error) {
	marshaledKey, err := dynamodbattribute.MarshalMap(k)
	if err != nil {
		return Item{}, err
	}

	out, err := db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		ConditionExpression: aws.String("#b = :b"),
		ExpressionAttributeNames: map[string]*string{
			"#b": aws.String("b"),
			"#a": aws.String("a"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":b": {S: aws.String(whenB)},
			":a": {S: aws.String(newA)},
		},
		Key:              marshaledKey,
		ReturnValues:     aws.String("ALL_NEW"),
		TableName:        aws.String(table),
		UpdateExpression: aws.String("REMOVE #b SET #a = :a"),
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

func PutIfNotExistsV1(ctx context.Context, db dynamodbiface.DynamoDBAPI, table string, k Key) error {
	marshaledKey, err := dynamodbattribute.MarshalMap(k)
	if err != nil {
		return err
	}

	_, err = db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		ConditionExpression: aws.String("attribute_not_exists(#pk)"),
		ExpressionAttributeNames: map[string]*string{
			"#pk": aws.String("pk"),
		},
		Item:      marshaledKey,
		TableName: aws.String(table),
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

