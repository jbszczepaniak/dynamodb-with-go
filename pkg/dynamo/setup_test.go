package dynamo_test

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

func TestSetupTable(t *testing.T) {
	ctx := context.Background()
	db, cleanup := dynamo.SetupTable(t, ctx, "PartitionKeyTable", "./testdata/template.yml")

	out, err := db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("PartitionKeyTable"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "PartitionKeyTable", *out.Table.TableName)
	assert.Equal(t, "pk", *out.Table.AttributeDefinitions[0].AttributeName)

	cleanup()
	_, err = db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("PartitionKeyTable"),
	})

	var notfound *types.ResourceNotFoundException
	assert.True(t, errors.As(err, &notfound))

}
