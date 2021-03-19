package episode10

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestSlicesMarshaling(t *testing.T) {

	t.Run("regular way", func(t *testing.T) {
		t.Skip("this test fails")
		attrs, err := attributevalue.Marshal([]string{})
		spew.Dump(attrs)
		assert.NoError(t, err)

		var s []string
		err = attributevalue.Unmarshal(attrs, &s)
		assert.NoError(t, err)

		assert.NotNil(t, s) // fails
		assert.Len(t, s, 0)
	})

	t.Run("new way", func(t *testing.T) {
		e := attributevalue.NewEncoder(func(opt *attributevalue.EncoderOptions) {
			opt.NullEmptySets = true
		})
		attrs, err := e.Encode([]string{})
		assert.NoError(t, err)

		var s []string
		err = attributevalue.Unmarshal(attrs, &s)
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.Len(t, s, 0)
	})
}
