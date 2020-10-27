package sensors_test

import (
	"context"
	"testing"
	"time"

	"dynamodb-with-go/episode8/v1/sensors"
	"dynamodb-with-go/pkg/dynamo"

	"github.com/stretchr/testify/assert"
)

func TestSensors(t *testing.T) {
	ctx := context.Background()

	sensor := sensors.Sensor{
		ID:       "sensor-1",
		City:     "Poznan",
		Building: "A",
		Floor:    "1",
		Room:     "123",
	}

	t.Run("register sensor, get sensor", func(t *testing.T) {
		tableName := "SensorsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
		defer cleanup()
		manager := sensors.NewManager(db, tableName)

		err := manager.Register(ctx, sensor)
		assert.NoError(t, err)

		returned, err := manager.Get(ctx, "sensor-1")
		assert.NoError(t, err)
		assert.Equal(t, sensor, returned)
	})

	t.Run("do not allow to register many times", func(t *testing.T) {
		tableName := "SensorsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
		defer cleanup()
		manager := sensors.NewManager(db, tableName)

		err := manager.Register(ctx, sensor)
		assert.NoError(t, err)

		err = manager.Register(ctx, sensor)
		assert.EqualError(t, err, "already registered")
	})

	t.Run("save new reading", func(t *testing.T) {
		tableName := "SensorsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
		defer cleanup()
		manager := sensors.NewManager(db, tableName)

		err := manager.Register(ctx, sensor)
		assert.NoError(t, err)

		err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.67", ReadAt: time.Now()})
		assert.NoError(t, err)

		_, latest, err := manager.LatestReadings(ctx, "sensor-1", 1)
		assert.NoError(t, err)
		assert.Equal(t, "0.67", latest[0].Value)
	})

	t.Run("get last readings and sensor", func(t *testing.T) {
		tableName := "SensorsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
		defer cleanup()
		manager := sensors.NewManager(db, tableName)

		err := manager.Register(ctx, sensor)

		assert.NoError(t, err)

		err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.3", ReadAt: time.Now().Add(-20 * time.Second)})
		assert.NoError(t, err)
		err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.5", ReadAt: time.Now().Add(-10 * time.Second)})
		assert.NoError(t, err)
		err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.67", ReadAt: time.Now()})
		assert.NoError(t, err)

		sensor, latest, err := manager.LatestReadings(ctx, "sensor-1", 2)
		assert.NoError(t, err)
		assert.Len(t, latest, 2)
		assert.Equal(t, "0.67", latest[0].Value)
		assert.Equal(t, "0.5", latest[1].Value)
		assert.Equal(t, "sensor-1", sensor.ID)
	})

	t.Run("get by sensors by location", func(t *testing.T) {
		tableName := "SensorsTable"
		db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
		defer cleanup()
		manager := sensors.NewManager(db, tableName)

		err := manager.Register(ctx, sensors.Sensor{ID: "sensor-1", City: "Poznan", Building: "A", Floor: "1", Room: "2"})
		err = manager.Register(ctx, sensors.Sensor{ID: "sensor-2", City: "Poznan", Building: "A", Floor: "2", Room: "4"})
		err = manager.Register(ctx, sensors.Sensor{ID: "sensor-3", City: "Poznan", Building: "A", Floor: "2", Room: "5"})

		ids, err := manager.GetSensors(ctx, sensors.Location{City: "Poznan", Building: "A", Floor: "2"})
		assert.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, "sensor-2")
		assert.Contains(t, ids, "sensor-3")
	})
}
