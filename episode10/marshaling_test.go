package episode10

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestSlicesMarshaling(t *testing.T) {

	t.Run("regular way", func(t *testing.T) {
		t.Skip("this test fails")
		attrs, err := dynamodbattribute.Marshal([]string{})
		spew.Dump(attrs)
		assert.NoError(t, err)

		var s []string
		err = dynamodbattribute.Unmarshal(attrs, &s)
		assert.NoError(t, err)

		assert.NotNil(t, s) // fails
		assert.Len(t, s, 0)
	})

	t.Run("new way", func(t *testing.T) {
		e := dynamodbattribute.NewEncoder(func(e *dynamodbattribute.Encoder) {
			e.EnableEmptyCollections = true
		})
		attrs, err := e.Encode([]string{})
		assert.NoError(t, err)

		var s []string
		d := dynamodbattribute.NewDecoder(func(d *dynamodbattribute.Decoder) {
			d.EnableEmptyCollections = true
		})
		err = d.Decode(attrs, &s)
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.Len(t, s, 0)
	})
}
