package dynamo

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/awslabs/goformation/cloudformation"
)

// FromCloudFormationToCreateInput transforms DynamoDB table from CloudFormation template
// into CreateTableInput struct, that can be used with aws-sdk-go to create the table.
func FromCloudFormationToCreateInput(t cloudformation.AWSDynamoDBTable) dynamodb.CreateTableInput {
	var input dynamodb.CreateTableInput
	for _, attrs := range t.AttributeDefinitions {
		input.AttributeDefinitions = append(input.AttributeDefinitions, &dynamodb.AttributeDefinition{
			AttributeName: aws.String(attrs.AttributeName),
			AttributeType: aws.String(attrs.AttributeType),
		})
	}
	for _, key := range t.KeySchema {
		input.KeySchema = append(input.KeySchema,  &dynamodb.KeySchemaElement{
			AttributeName: aws.String(key.AttributeName),
			KeyType:       aws.String(key.KeyType),
		})
	}
	for _, idx := range t.LocalSecondaryIndexes {
		if idx.Projection.ProjectionType == "INCLUDE" {
			panic("not implemented")
		}
		indexKeySchema := []*dynamodb.KeySchemaElement{}
		for _, key := range idx.KeySchema {
			indexKeySchema = append(indexKeySchema, &dynamodb.KeySchemaElement{
				AttributeName: aws.String(key.AttributeName),
				KeyType:       aws.String(key.KeyType),
			})
		}
		input.LocalSecondaryIndexes = append(input.LocalSecondaryIndexes, &dynamodb.LocalSecondaryIndex{
			IndexName:  aws.String(idx.IndexName),
			KeySchema:  indexKeySchema,
			Projection: &dynamodb.Projection{ProjectionType: aws.String(idx.Projection.ProjectionType)},
		})
	}

	input.TableName = aws.String(t.TableName)
	input.BillingMode = aws.String(t.BillingMode)
	return input
}