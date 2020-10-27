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

	t.Run("table with single local secondary indexes", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndLocalIndexTable")
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
				{
					AttributeName: aws.String("lsi_sk"),
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
			LocalSecondaryIndexes: []*dynamodb.LocalSecondaryIndex{
				{
					IndexName: aws.String("MyIndex"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("pk"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("lsi_sk"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
			},
			TableName:              aws.String("CompositePrimaryKeyAndLocalIndexTable"),
		}, input)
	})

	t.Run("table with many local secondary indexes", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndManyLocalIndexesTable")
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
				{
					AttributeName: aws.String("lsi1_sk"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("lsi2_sk"),
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
			LocalSecondaryIndexes: []*dynamodb.LocalSecondaryIndex{
				{
					IndexName: aws.String("MyIndex1"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("pk"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("lsi1_sk"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
				{
					IndexName: aws.String("MyIndex2"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("pk"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("lsi2_sk"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
			},
			TableName:              aws.String("CompositePrimaryKeyAndManyLocalIndexesTable"),
		}, input)
	})

	t.Run("table with single global secondary index", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndSingleGlobalIndexTable")
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
				{
					AttributeName: aws.String("gsi1_pk"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("gsi1_sk"),
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
			GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
				{
					IndexName: aws.String("GlobalSecondaryIndex1"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("gsi1_pk"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("gsi1_sk"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},	
				},
			},
			TableName:              aws.String("CompositePrimaryKeyAndSingleGlobalIndexTable"),
		}, input)
	})

	t.Run("table with many global secondary indexes", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndManyGlobalIndexTable")
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
				{
					AttributeName: aws.String("gsi1_pk"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("gsi1_sk"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("gsi2_pk"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("gsi2_sk"),
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
			GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
				{
					IndexName: aws.String("GlobalSecondaryIndex1"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("gsi1_pk"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("gsi1_sk"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},	
				},
				{
					IndexName: aws.String("GlobalSecondaryIndex2"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("gsi2_pk"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("gsi2_sk"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},	
				},
			},
			TableName:              aws.String("CompositePrimaryKeyAndManyGlobalIndexTable"),
		}, input)
	})
}
