package episode12

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/stretchr/testify/assert"
)

func insert(ctx context.Context, db *dynamodb.Client, table string, items ...Item) {
	for _, i := range items {
		attrs, err := attributevalue.MarshalMap(i)
		if err != nil {
			panic(err)
		}

		_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
			Item:      attrs,
			TableName: aws.String(table),
		})
		if err != nil {
			panic(err)
		}
	}
}

func TestExpressions(t *testing.T) {
	item1 := Item{Key: Key{PK: "1", SK: "1"}, A: "foo", B: "bar"}
	item2 := Item{Key: Key{PK: "1", SK: "2"}, A: "foo", B: "baz"}

	t.Run("v1 - get whole item collection", func(t *testing.T) {
		ctx := context.Background()
		tableName := "ATable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()
		insert(ctx, db, tableName, item1, item2)

		collection, err := GetItemCollectionV1(ctx, db, tableName, "1")
		assert.NoError(t, err)
		assert.Subset(t, collection, []Item{item1, item2})
		assert.Len(t, collection, 2)
	})

	t.Run("v2 - get whole item collection", func(t *testing.T) {
		ctx := context.Background()
		tableName := "ATable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()
		insert(ctx, db, tableName, item1, item2)

		collection, err := GetItemCollectionV2(ctx, db, tableName, "1")
		assert.NoError(t, err)
		assert.Subset(t, collection, []Item{item1, item2})
		assert.Len(t, collection, 2)
	})

	t.Run("v1 - update A and unset B but only if B is set to `baz`", func(t *testing.T) {
		ctx := context.Background()
		tableName := "ATable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()
		insert(ctx, db, tableName, item1, item2)

		updated, err := UpdateAWhenBAndUnsetBV1(ctx, db, tableName, Key{PK: "1", SK: "1"}, "newA", "baz")
		if assert.Error(t, err) {
			assert.Equal(t, "b is not baz, aborting update", err.Error())
		}
		assert.Empty(t, updated)

		updated, err = UpdateAWhenBAndUnsetBV1(ctx, db, tableName, Key{PK: "1", SK: "2"}, "newA", "baz")
		assert.NoError(t, err)
		assert.Equal(t, "newA", updated.A)
		assert.Empty(t, updated.B)
	})

	t.Run("v2 - update A and unset B but only if B is set to `baz`", func(t *testing.T) {
		ctx := context.Background()
		tableName := "ATable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()
		insert(ctx, db, tableName, item1, item2)

		updated, err := UpdateAWhenBAndUnsetBV2(ctx, db, tableName, Key{PK: "1", SK: "1"}, "newA", "baz")
		if assert.Error(t, err) {
			assert.Equal(t, "b is not baz, aborting update", err.Error())
		}
		assert.Empty(t, updated)

		updated, err = UpdateAWhenBAndUnsetBV2(ctx, db, tableName, Key{PK: "1", SK: "2"}, "newA", "baz")
		assert.NoError(t, err)
		assert.Equal(t, "newA", updated.A)
		assert.Empty(t, updated.B)
	})

	t.Run("v1 - put if doesn't exist", func(t *testing.T) {
		ctx := context.Background()
		tableName := "ATable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()
		insert(ctx, db, tableName, item1, item2)

		err := PutIfNotExistsV1(ctx, db, tableName, Key{PK: "1", SK: "2"})
		if assert.Error(t, err) {
			assert.Equal(t, "Item with this Key already exists", err.Error())
		}

		err = PutIfNotExistsV1(ctx, db, tableName, Key{PK: "10", SK: "20"})
		assert.NoError(t, err)
	})

	t.Run("v2 - put if doesn't exist", func(t *testing.T) {
		ctx := context.Background()
		tableName := "ATable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()
		insert(ctx, db, tableName, item1, item2)

		err := PutIfNotExistsV2(ctx, db, tableName, Key{PK: "1", SK: "2"})
		if assert.Error(t, err) {
			assert.Equal(t, "Item with this Key already exists", err.Error())
		}

		err = PutIfNotExistsV2(ctx, db, tableName, Key{PK: "10", SK: "20"})
		assert.NoError(t, err)

	})
}
