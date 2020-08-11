package dynamo

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestSetupTable(t *testing.T) {
	ctx := context.Background()
	db, cleanup := SetupTable(t, ctx, "PartitionKeyTable", "./testdata/template.yml")

	out, err := db.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("PartitionKeyTable"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "PartitionKeyTable", aws.StringValue(out.Table.TableName))
	assert.Equal(t, "pk", aws.StringValue(out.Table.AttributeDefinitions[0].AttributeName))

	cleanup()
	_, err = db.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("PartitionKeyTable"),
	})
	aerr, ok := err.(awserr.Error)
	assert.True(t, ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException)
}
