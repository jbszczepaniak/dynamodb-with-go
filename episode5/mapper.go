package episode5

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/google/uuid"
)

// Mapper keeps Dynamo dependency.
type Mapper struct {
	db    dynamodbiface.DynamoDBAPI
	table string
}

// NewMapper creates instance of Mapper.
func NewMapper(client dynamodbiface.DynamoDBAPI, table string) *Mapper {
	return &Mapper{db: client, table: table}
}

type mapping struct {
	OldID string `dynamodbav:"old_id"`
	NewID string `dynamodbav:"new_id"`
}

// Map generates new ID for old ID or retrieves already created new ID.
func (m *Mapper) Map(ctx context.Context, old string) (string, error) {
	idsMapping := mapping{OldID: old, NewID: uuid.New().String()}
	attrs, err := dynamodbattribute.MarshalMap(&idsMapping)
	if err != nil {
		return "", err
	}

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeNotExists(expression.Name("old_id"))).
		Build()
	if err != nil {
		return "", err
	}

	_, err = m.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Item:                      attrs,
		TableName:                 aws.String(m.table),
	})
	if err == nil {
		return idsMapping.NewID, nil
	}
	aerr, ok := err.(awserr.Error)
	if ok && aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
		return "", err
	}

	out, err := m.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"old_id": {S: aws.String(old)},
		},
		TableName: aws.String(m.table),
	})
	if err != nil {
		return "", err
	}
	return aws.StringValue(out.Item["new_id"].S), nil
}
