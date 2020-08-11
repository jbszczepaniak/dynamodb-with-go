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
		input.KeySchema = append(input.KeySchema, &dynamodb.KeySchemaElement{
			AttributeName: aws.String(key.AttributeName),
			KeyType:       aws.String(key.KeyType),
		})
	}
	input.TableName = aws.String(t.TableName)
	input.BillingMode = aws.String(t.BillingMode)
	return input
}