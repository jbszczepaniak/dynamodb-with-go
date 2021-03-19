package episode12

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func GetItemCollectionV1(ctx context.Context, db *dynamodb.Client, table, pk string) ([]Item, error) {
	out, err := db.Query(ctx, &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("#key = :value"),
		ExpressionAttributeNames: map[string]string{
			"#key": "pk",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":value": &types.AttributeValueMemberS{Value: pk},
		},
		TableName: aws.String(table),
	})
	if err != nil {
		return nil, err
	}

	var items []Item
	err = attributevalue.UnmarshalListOfMaps(out.Items, &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func UpdateAWhenBAndUnsetBV1(ctx context.Context, db *dynamodb.Client, table string, k Key, newA, whenB string) (Item, error) {
	marshaledKey, err := attributevalue.MarshalMap(k)
	if err != nil {
		return Item{}, err
	}

	out, err := db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		ConditionExpression: aws.String("#b = :b"),
		ExpressionAttributeNames: map[string]string{
			"#b": "b",
			"#a": "a",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":b": &types.AttributeValueMemberS{Value: whenB},
			":a": &types.AttributeValueMemberS{Value: newA},
		},
		Key:              marshaledKey,
		ReturnValues:     types.ReturnValueAllNew,
		TableName:        aws.String(table),
		UpdateExpression: aws.String("REMOVE #b SET #a = :a"),
	})
	if err != nil {
		var conditionFailed *types.ConditionalCheckFailedException
		if errors.As(err, &conditionFailed) {
			return Item{}, fmt.Errorf("b is not %s, aborting update", whenB)
		}
		return Item{}, err
	}
	var i Item
	err = attributevalue.UnmarshalMap(out.Attributes, &i)
	if err != nil {
		return Item{}, err
	}
	return i, nil
}

func PutIfNotExistsV1(ctx context.Context, db *dynamodb.Client, table string, k Key) error {
	marshaledKey, err := attributevalue.MarshalMap(k)
	if err != nil {
		return err
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		ConditionExpression: aws.String("attribute_not_exists(#pk)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": "pk",
		},
		Item:      marshaledKey,
		TableName: aws.String(table),
	})

	if err != nil {
		var conditionFailed *types.ConditionalCheckFailedException
		if errors.As(err, &conditionFailed) {
			return errors.New("Item with this Key already exists")
		}
		return err
	}

	return nil
}
