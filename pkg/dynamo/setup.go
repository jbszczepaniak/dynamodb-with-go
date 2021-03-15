package dynamo

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/awslabs/goformation"
)

type EndpointResolver struct{}

func (e EndpointResolver) ResolveEndpoint(region string, options dynamodb.EndpointResolverOptions) (aws.Endpoint, error) {
	return aws.Endpoint{URL: "http://localhost:8000"}, nil
}

func localDynamoDB(t *testing.T) *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("local"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "local")),
	)
	if err != nil {
		t.Fatal("could not setup db connection")
	}

	db := dynamodb.NewFromConfig(cfg, dynamodb.WithEndpointResolver(EndpointResolver{}))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err = db.ListTables(ctx, nil)
	if err != nil {
		t.Fatal("make sure DynamoDB local runs on port :8000", err)
	}
	return db
}

// SetupTable creates table defined in the CloudFormation template file under `path`.
// It returns connection to the DynamoDB and cleanup function, that needs to be run after tests.
func SetupTable(t *testing.T, ctx context.Context, tableName, path string) (*dynamodb.Client, func()) {
	db := localDynamoDB(t)
	tmpl, err := goformation.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	table, err := tmpl.GetAWSDynamoDBTableWithName(tableName)
	if err != nil {
		t.Fatal(err)
	}
	input := FromCloudFormationToCreateInput(*table)
	_, err = db.CreateTable(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}
	return db, func() {
		db.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(tableName)})
	}
}
