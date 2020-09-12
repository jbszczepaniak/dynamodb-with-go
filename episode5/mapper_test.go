package episode5

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapping(t *testing.T) {
	t.Run("generate new ID for each legacy ID", func(t *testing.T) {
		ctx := context.Background()
		tableName := "LegacyIDsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()

		mapper := NewMapper(db, tableName)

		first, err := mapper.Map(ctx, "123")
		assert.NoError(t, err)
		assert.NotEmpty(t, first)
		assert.NotEqual(t, "123", first)

		second, err := mapper.Map(ctx, "456")
		assert.NoError(t, err)
		assert.NotEmpty(t, second)
		assert.NotEqual(t, "456", second)

		assert.NotEqual(t, first, second)
	})

	t.Run("do not regenerate ID for the same legacy ID", func(t *testing.T) {
		ctx := context.Background()
		tableName := "LegacyIDsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()

		mapper := NewMapper(db, tableName)

		first, err := mapper.Map(ctx, "123")
		assert.NoError(t, err)

		second, err := mapper.Map(ctx, "123")
		assert.NoError(t, err)

		assert.Equal(t, first, second)
	})

}
