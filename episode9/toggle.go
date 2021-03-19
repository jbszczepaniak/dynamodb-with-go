package episode9

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Toggle struct {
	db    *dynamodb.Client
	table string
}

type Switch struct {
	ID        string
	State     bool
	CreatedAt time.Time
}

type switchItem struct {
	PK string `dynamodbav:"pk"`
	SK string `dynamodbav:"sk"`

	State     bool      `dynamodbav:"state"`
	CreatedAt time.Time `dynamodbav:"created_at"`
}

func (s switchItem) asSwitch() Switch {
	return Switch{
		ID:        s.PK,
		State:     s.State,
		CreatedAt: s.CreatedAt,
	}
}

func (s Switch) asLogItem() switchItem {
	return switchItem{
		PK:        s.ID,
		SK:        "SWITCH#" + s.CreatedAt.Format(time.RFC3339Nano),
		CreatedAt: s.CreatedAt,
		State:     s.State,
	}
}

func (s Switch) asLatestItem() switchItem {
	return switchItem{
		PK:        s.ID,
		SK:        "LATEST_SWITCH",
		CreatedAt: s.CreatedAt,
		State:     s.State,
	}
}

func NewToggle(db *dynamodb.Client, table string) *Toggle {
	return &Toggle{db: db, table: table}
}

func (t *Toggle) Save(ctx context.Context, s Switch) error {
	item := s.asLogItem()
	attrs, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}

	expr, err := expression.NewBuilder().
		WithCondition(expression.LessThan(expression.Name("created_at"), expression.Value(item.CreatedAt))).
		WithUpdate(expression.
			Set(expression.Name("created_at"), expression.Value(item.CreatedAt)).
			Set(expression.Name("state"), expression.Value(item.State))).
		Build()

	if err != nil {
		return err
	}
	_, err = t.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Update: &types.Update{
					Key: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: item.PK},
						"sk": &types.AttributeValueMemberS{Value: "LATEST_SWITCH"},
					},
					ExpressionAttributeNames:            expr.Names(),
					ExpressionAttributeValues:           expr.Values(),
					ConditionExpression:                 expr.Condition(),
					TableName:                           aws.String(t.table),
					UpdateExpression:                    expr.Update(),
					ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailure(types.ReturnValueAllOld),
				},
			},
			{
				Put: &types.Put{
					Item:      attrs,
					TableName: aws.String(t.table),
				},
			},
		},
	})

	if err == nil {
		return nil
	}
	var transactionCanelled *types.TransactionCanceledException
	if !errors.As(err, &transactionCanelled) {
		return err
	}

	if len(transactionCanelled.CancellationReasons[0].Item) > 0 {
		return nil
	}

	expr, err = expression.NewBuilder().
		WithCondition(
			expression.Not(expression.And(
				expression.Equal(expression.Name("pk"), expression.Value(item.PK)),
				expression.Equal(expression.Name("sk"), expression.Value("LATEST_SWITCH")),
			))).Build()
	if err != nil {
		return err
	}

	latestAttrs, err := attributevalue.MarshalMap(s.asLatestItem())

	_, err = t.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeNames:  expr.Names(),
					ExpressionAttributeValues: expr.Values(),
					Item:                      latestAttrs,
					TableName:                 aws.String(t.table),
				},
			},
			{
				Put: &types.Put{
					Item:      attrs,
					TableName: aws.String(t.table),
				},
			},
		},
	})
	if err == nil {
		return nil
	}
	if !errors.As(err, &transactionCanelled) {
		return err
	}

	return t.Save(ctx, s)
}

func (t *Toggle) Latest(ctx context.Context, userID string) (Switch, error) {
	out, err := t.db.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: userID},
			"sk": &types.AttributeValueMemberS{Value: "LATEST_SWITCH"},
		},
		TableName: aws.String(t.table),
	})
	if err != nil {
		return Switch{}, err
	}
	if len(out.Item) == 0 {
		return Switch{}, errors.New("not found")
	}

	var item switchItem
	err = attributevalue.UnmarshalMap(out.Item, &item)
	return item.asSwitch(), err
}
