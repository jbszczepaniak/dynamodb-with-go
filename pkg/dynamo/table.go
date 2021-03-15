package dynamo

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/awslabs/goformation/cloudformation"
)

// FromCloudFormationToCreateInput transforms DynamoDB table from CloudFormation template
// into CreateTableInput struct, that can be used with aws-sdk-go to create the table.
func FromCloudFormationToCreateInput(t cloudformation.AWSDynamoDBTable) dynamodb.CreateTableInput {
	var input dynamodb.CreateTableInput
	for _, attrs := range t.AttributeDefinitions {
		input.AttributeDefinitions = append(input.AttributeDefinitions, types.AttributeDefinition{
			AttributeName: aws.String(attrs.AttributeName),
			AttributeType: types.ScalarAttributeType(attrs.AttributeType),
		})
	}
	for _, key := range t.KeySchema {
		input.KeySchema = append(input.KeySchema, types.KeySchemaElement{
			AttributeName: aws.String(key.AttributeName),
			KeyType:       types.KeyType(key.KeyType),
		})
	}
	for _, idx := range t.LocalSecondaryIndexes {
		if idx.Projection.ProjectionType != "ALL" {
			panic("not implemented")
		}
		indexKeySchema := []types.KeySchemaElement{}
		for _, key := range idx.KeySchema {
			indexKeySchema = append(indexKeySchema, types.KeySchemaElement{
				AttributeName: aws.String(key.AttributeName),
				KeyType:       types.KeyType(key.KeyType),
			})
		}
		input.LocalSecondaryIndexes = append(input.LocalSecondaryIndexes, types.LocalSecondaryIndex{
			IndexName:  aws.String(idx.IndexName),
			KeySchema:  indexKeySchema,
			Projection: &types.Projection{ProjectionType: types.ProjectionType(idx.Projection.ProjectionType)},
		})
	}
	for _, idx := range t.GlobalSecondaryIndexes {
		if idx.Projection.ProjectionType != "ALL" {
			panic("not implemented")
		}
		indexKeySchema := []types.KeySchemaElement{}
		for _, key := range idx.KeySchema {
			indexKeySchema = append(indexKeySchema, types.KeySchemaElement{
				AttributeName: aws.String(key.AttributeName),
				KeyType:       types.KeyType(key.KeyType),
			})
		}
		input.GlobalSecondaryIndexes = append(input.GlobalSecondaryIndexes, types.GlobalSecondaryIndex{
			IndexName:  aws.String(idx.IndexName),
			KeySchema:  indexKeySchema,
			Projection: &types.Projection{ProjectionType: types.ProjectionType(idx.Projection.ProjectionType)},
		})
	}

	input.TableName = aws.String(t.TableName)
	input.BillingMode = types.BillingMode(t.BillingMode)
	return input
}
