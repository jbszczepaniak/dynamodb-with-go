package episode6

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// Mapper keeps Dynamo dependency.
type Mapper struct {
	db    *dynamodb.Client
	table string
}

// NewMapper creates instance of Mapper.
func NewMapper(client *dynamodb.Client, table string) *Mapper {
	return &Mapper{db: client, table: table}
}

type mapping struct {
	OldID string `dynamodbav:"old_id"`
	NewID string `dynamodbav:"new_id"`
}

// Map generates new ID for old ID or retrieves already created new ID.
func (m *Mapper) Map(ctx context.Context, old string) (string, error) {
	idsMapping := mapping{OldID: old, NewID: uuid.New().String()}
	attrs, err := attributevalue.MarshalMap(&idsMapping)
	if err != nil {
		return "", err
	}

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeNotExists(expression.Name("old_id"))).
		Build()
	if err != nil {
		return "", err
	}

	_, err = m.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{

		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					ConditionExpression:                 expr.Condition(),
					ExpressionAttributeNames:            expr.Names(),
					ExpressionAttributeValues:           expr.Values(),
					Item:                                attrs,
					ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailure(types.ReturnValueAllOld),
					TableName:                           aws.String(m.table),
				},
			},
		},
	})

	if err == nil {
		return idsMapping.NewID, nil
	}

	var transactionCanelled *types.TransactionCanceledException
	if !errors.As(err, &transactionCanelled) {
		return "", err
	}

	// ALL_OLD is not empty - mapping exists.
	if len(transactionCanelled.CancellationReasons[0].Item) > 0 {
		var ret mapping
		attributevalue.UnmarshalMap(transactionCanelled.CancellationReasons[0].Item, &ret)
		return ret.NewID, nil
	}

	return "", err
}
