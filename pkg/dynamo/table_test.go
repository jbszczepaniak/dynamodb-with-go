package dynamo_test

import (
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
			},
			TableName: aws.String("PartitionKeyTable"),
		}, input)
	})

	t.Run("table with composite primary key", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       types.KeyTypeRange,
				},
			},
			TableName: aws.String("CompositePrimaryKeyTable"),
		}, input)
	})

	t.Run("table with single local secondary indexes", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndLocalIndexTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("lsi_sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       types.KeyTypeRange,
				},
			},
			LocalSecondaryIndexes: []types.LocalSecondaryIndex{
				{
					IndexName: aws.String("MyIndex"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("pk"),
							KeyType:       types.KeyTypeHash,
						},
						{
							AttributeName: aws.String("lsi_sk"),
							KeyType:       types.KeyTypeRange,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
			},
			TableName: aws.String("CompositePrimaryKeyAndLocalIndexTable"),
		}, input)
	})

	t.Run("table with many local secondary indexes", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndManyLocalIndexesTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("lsi1_sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("lsi2_sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       types.KeyTypeRange,
				},
			},
			LocalSecondaryIndexes: []types.LocalSecondaryIndex{
				{
					IndexName: aws.String("MyIndex1"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("pk"),
							KeyType:       types.KeyTypeHash,
						},
						{
							AttributeName: aws.String("lsi1_sk"),
							KeyType:       types.KeyTypeRange,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
				{
					IndexName: aws.String("MyIndex2"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("pk"),
							KeyType:       types.KeyTypeHash,
						},
						{
							AttributeName: aws.String("lsi2_sk"),
							KeyType:       types.KeyTypeRange,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
			},
			TableName: aws.String("CompositePrimaryKeyAndManyLocalIndexesTable"),
		}, input)
	})

	t.Run("table with single global secondary index", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndSingleGlobalIndexTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("gsi1_pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("gsi1_sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       types.KeyTypeRange,
				},
			},
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
				{
					IndexName: aws.String("GlobalSecondaryIndex1"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("gsi1_pk"),
							KeyType:       types.KeyTypeHash,
						},
						{
							AttributeName: aws.String("gsi1_sk"),
							KeyType:       types.KeyTypeRange,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
			},
			TableName: aws.String("CompositePrimaryKeyAndSingleGlobalIndexTable"),
		}, input)
	})

	t.Run("table with many global secondary indexes", func(t *testing.T) {
		tmpl, err := goformation.Open("./testdata/template.yml")
		assert.NoError(t, err)

		table, err := tmpl.GetAWSDynamoDBTableWithName("CompositePrimaryKeyAndManyGlobalIndexTable")
		assert.NoError(t, err)

		input := dynamo.FromCloudFormationToCreateInput(*table)
		assert.Equal(t, dynamodb.CreateTableInput{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("gsi1_pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("gsi1_sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("gsi2_pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("gsi2_sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       types.KeyTypeRange,
				},
			},
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
				{
					IndexName: aws.String("GlobalSecondaryIndex1"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("gsi1_pk"),
							KeyType:       types.KeyTypeHash,
						},
						{
							AttributeName: aws.String("gsi1_sk"),
							KeyType:       types.KeyTypeRange,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
				{
					IndexName: aws.String("GlobalSecondaryIndex2"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("gsi2_pk"),
							KeyType:       types.KeyTypeHash,
						},
						{
							AttributeName: aws.String("gsi2_sk"),
							KeyType:       types.KeyTypeRange,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
			},
			TableName: aws.String("CompositePrimaryKeyAndManyGlobalIndexTable"),
		}, input)
	})
}
