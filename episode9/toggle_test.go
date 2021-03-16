package episode9

import (
	"context"
	"dynamodb-with-go/pkg/dynamo"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToggle(t *testing.T) {
	ctx := context.Background()

	t.Run("save toggle", func(t *testing.T) {
		tableName := "ToggleStateTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()

		toggle := NewToggle(db, tableName)
		err := toggle.Save(ctx, Switch{ID: "123", State: true, CreatedAt: time.Now()})
		assert.NoError(t, err)

		s, err := toggle.Latest(ctx, "123")
		assert.NoError(t, err)
		assert.Equal(t, s.State, true)
	})

	t.Run("save toggles, retrieve latest", func(t *testing.T) {
		tableName := "ToggleStateTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()

		toggle := NewToggle(db, tableName)
		now := time.Now()
		err := toggle.Save(ctx, Switch{ID: "123", State: true, CreatedAt: now})
		assert.NoError(t, err)

		err = toggle.Save(ctx, Switch{ID: "123", State: false, CreatedAt: now.Add(10 * time.Second)})
		assert.NoError(t, err)

		s, err := toggle.Latest(ctx, "123")
		assert.NoError(t, err)
		assert.Equal(t, s.State, false)
	})

	t.Run("drop out of order switch", func(t *testing.T) {
		tableName := "ToggleStateTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
		defer cleanup()

		toggle := NewToggle(db, tableName)
		now := time.Now()
		err := toggle.Save(ctx, Switch{ID: "123", State: true, CreatedAt: now})
		assert.NoError(t, err)

		err = toggle.Save(ctx, Switch{ID: "123", State: false, CreatedAt: now.Add(-10 * time.Second)})
		assert.NoError(t, err)

		s, err := toggle.Latest(ctx, "123")
		assert.NoError(t, err)
		assert.Equal(t, s.State, true)
	})
}
