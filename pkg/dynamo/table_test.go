package dynamo_test

import (
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/awslabs/goformation"
	"github.com/stretchr/testify/assert"
)

func TestTable(t *testing.T) {
	t.Run("table with partition key only", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("PartitionKeyTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions:   []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: aws.String("S"),
				},
			},
			BillingMode:            aws.String("PAY_PER_REQUEST"),
			KeySchema:              []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       aws.String("HASH"),
				},
			},
			TableName:              aws.String("PartitionKeyTable"),
		}, input)
	})

	t.Run("table with composite primary key", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions:   []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: aws.String("S"),
				},
			},
			BillingMode:            aws.String("PAY_PER_REQUEST"),
			KeySchema:              []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       aws.String("RANGE"),
				},
			},
			TableName:              aws.String("CompositePrimaryKeyTable"),
		}, input)
	})
}
