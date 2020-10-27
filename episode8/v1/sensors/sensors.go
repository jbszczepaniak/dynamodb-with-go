package sensors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type Sensor struct {
	ID       string
	City     string
	Building string
	Floor    string
	Room     string
}

type Reading struct {
	SensorID string
	Value    string
	ReadAt   time.Time
}

type Location struct {
	City     string
	Building string
	Floor    string
}

func (l Location) asPath() string {
	path := "LOCATION#"
	if l.Building == "" {
		return path
	}
	path = path + l.Building + "#"
	if l.Floor == "" {
		return path
	}
	return path + l.Floor
}

func (s Sensor) asItem() sensorItem {
	return sensorItem{
		City:     s.City,
		ID:       "SENSOR#" + s.ID,
		SK:       "SENSORINFO",
		Building: s.Building,
		Floor:    s.Floor,
		Room:     s.Room,
	}
}

type sensorItem struct {
	ID string `dynamodbav:"pk"`
	SK string `dynamodbav:"sk"`

	City     string `dynamodbav:"city"`
	Building string `dynamodbav:"building"`
	Floor    string `dynamodbav:"floor"`
	Room     string `dynamodbav:"room"`
}

type readingItem struct {
	SensorID string `dynamodbav:"pk"`
	Value    string `dynamodbav:"value"`
	ReadAt   string `dynamodbav:"sk"`
}

func (r Reading) asItem() readingItem {
	return readingItem{
		SensorID: "SENSOR#" + r.SensorID,
		ReadAt:   "READ#" + r.ReadAt.Format(time.RFC3339),
		Value:    r.Value,
	}
}

func (si sensorItem) asSensor() Sensor {
	return Sensor{
		ID:       strings.Split(si.ID, "#")[1],
		City:     si.City,
		Building: si.Building,
		Floor:    si.Floor,
		Room:     si.Room,
	}
}

func (ri readingItem) asReading() Reading {
	t, err := time.Parse(time.RFC3339, strings.Split(ri.ReadAt, "#")[1])
	if err != nil {
		panic("I would handle that in production")
	}
	return Reading{
		SensorID: strings.Split(ri.SensorID, "#")[1],
		ReadAt:   t,
		Value:    ri.Value,
	}
}

func NewManager(db dynamodbiface.DynamoDBAPI, table string) *sensorManager {
	return &sensorManager{db: db, table: table}
}

type SensorsManager interface {
	Register(ctx context.Context, sensor Sensor) error
	Get(ctx context.Context, id string) (Sensor, error)
}

type sensorManager struct {
	db    dynamodbiface.DynamoDBAPI
	table string
}

func (s *sensorManager) Register(ctx context.Context, sensor Sensor) error {
	attrs, err := dynamodbattribute.MarshalMap(sensor.asItem())
	if err != nil {
		return err
	}
	expr, err := expression.NewBuilder().WithCondition(expression.AttributeNotExists(expression.Name("pk"))).Build()
	if err != nil {
		return err
	}

	_, err = s.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Put: &dynamodb.Put{
					ConditionExpression:       expr.Condition(),
					ExpressionAttributeNames:  expr.Names(),
					ExpressionAttributeValues: expr.Values(),

					Item:      attrs,
					TableName: aws.String(s.table),
				},
			},
			{
				Put: &dynamodb.Put{
					Item: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String("CITY#" + sensor.City)},
						"sk": {S: aws.String(fmt.Sprintf("LOCATION#%s#%s#%s", sensor.Building, sensor.Floor, sensor.Room))},
						"id": {S: aws.String(sensor.ID)},
					},
					TableName: aws.String(s.table),
				},
			},
		},
	})
	if err != nil {
		_, ok := err.(*dynamodb.TransactionCanceledException)
		if ok {
			return errors.New("already registered")
		}
		return err
	}
	return nil
}

func (s *sensorManager) Get(ctx context.Context, id string) (Sensor, error) {
	out, err := s.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String("SENSOR#" + id)},
			"sk": {S: aws.String("SENSORINFO")},
		},
		TableName: aws.String(s.table),
	})
	if err != nil {
		return Sensor{}, err
	}

	var si sensorItem
	err = dynamodbattribute.UnmarshalMap(out.Item, &si)
	if err != nil {
		return Sensor{}, err
	}
	return si.asSensor(), nil
}

func (s *sensorManager) SaveReading(ctx context.Context, reading Reading) error {
	attrs, err := dynamodbattribute.MarshalMap(reading.asItem())
	if err != nil {
		return err
	}
	_, err = s.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		Item:      attrs,
		TableName: aws.String(s.table),
	})
	return err
}

func (s *sensorManager) LatestReadings(ctx context.Context, sensorID string, last int64) (Sensor, []Reading, error) {
	expr, err := expression.NewBuilder().WithKeyCondition(expression.KeyAnd(
		expression.KeyEqual(expression.Key("pk"), expression.Value("SENSOR#"+sensorID)),
		expression.KeyLessThanEqual(expression.Key("sk"), expression.Value("SENSORINFO")),
	)).Build()
	if err != nil {
		return Sensor{}, nil, err
	}

	out, err := s.db.QueryWithContext(ctx, &dynamodb.QueryInput{
		ExpressionAttributeValues: expr.Values(),
		ExpressionAttributeNames:  expr.Names(),
		KeyConditionExpression:    expr.KeyCondition(),
		Limit:                     aws.Int64(last + 1),
		ScanIndexForward:          aws.Bool(false),
		TableName:                 aws.String(s.table),
	})
	if err != nil {
		return Sensor{}, nil, err
	}

	var si sensorItem
	err = dynamodbattribute.UnmarshalMap(out.Items[0], &si)

	var ri []readingItem
	err = dynamodbattribute.UnmarshalListOfMaps(out.Items[1:aws.Int64Value(out.Count)], &ri)

	var readings []Reading
	for _, r := range ri {
		readings = append(readings, r.asReading())
	}
	return si.asSensor(), readings, nil
}

func (s *sensorManager) GetSensors(ctx context.Context, location Location) ([]string, error) {
	expr, err := expression.NewBuilder().WithKeyCondition(expression.KeyAnd(
		expression.KeyEqual(expression.Key("pk"), expression.Value("CITY#"+location.City)),
		expression.KeyBeginsWith(expression.Key("sk"), location.asPath()),
	)).Build()
	if err != nil {
		return nil, err
	}

	out, err := s.db.QueryWithContext(ctx, &dynamodb.QueryInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		TableName:                 aws.String(s.table),
	})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, i := range out.Items {
		ids = append(ids, aws.StringValue(i["id"].S))
	}
	return ids, nil
}
