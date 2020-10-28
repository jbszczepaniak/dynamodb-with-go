package episode10

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type Toggle struct {
	db    dynamodbiface.DynamoDBAPI
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


func NewToggle(db dynamodbiface.DynamoDBAPI, table string) *Toggle {
	return &Toggle{db: db, table: table}
}

func (t *Toggle) Save(ctx context.Context, s Switch) error {
	item := s.asLogItem()
	attrs, err := dynamodbattribute.MarshalMap(item)
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
	_, err = t.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Update: &dynamodb.Update{
					Key: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String(item.PK)},
						"sk": {S: aws.String("LATEST_SWITCH")},
					},
					ExpressionAttributeNames:  expr.Names(),
					ExpressionAttributeValues: expr.Values(),
					ConditionExpression:       expr.Condition(),
					TableName:                 aws.String(t.table),
					UpdateExpression:          expr.Update(),
					ReturnValuesOnConditionCheckFailure: aws.String("ALL_OLD"),
				},
			},
			{
				Put: &dynamodb.Put{
					Item:      attrs,
					TableName: aws.String(t.table),
				},
			},
		},
	})

	if err == nil {
		return nil
	}

	aerr, ok := err.(*dynamodb.TransactionCanceledException)
	if !ok {
		return err
	}

	if len(aerr.CancellationReasons[0].Item) > 0 {
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

	latestAttrs, err := dynamodbattribute.MarshalMap(s.asLatestItem())

	_, err = t.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Put: &dynamodb.Put{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeNames:  expr.Names(),
					ExpressionAttributeValues: expr.Values(),
					Item:                      latestAttrs,
					TableName:                 aws.String(t.table),
				},
			},
			{
				Put: &dynamodb.Put{
					Item:      attrs,
					TableName: aws.String(t.table),
				},
			},
		},
	})
	if err == nil {
		return nil
	}
	_, ok = err.(*dynamodb.TransactionCanceledException)
	if !ok {
		return err
	}

	return t.Save(ctx, s)
}

func (t *Toggle) Latest(ctx context.Context, userID string) (Switch, error) {
	out, err := t.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(userID)},
			"sk": {S: aws.String("LATEST_SWITCH")},
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
	err = dynamodbattribute.UnmarshalMap(out.Item, &item)
	return item.asSwitch(), err
}
