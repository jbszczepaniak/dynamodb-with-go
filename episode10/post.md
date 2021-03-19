# DynamoDB with Go #10

This time we are going to cover the gotcha of AWS SDK for Go that has bitten me recently. It's about storing empty slices in the DynamoDB.

## [Different flavours of empty slices](#different-falvors-of-empty-slices)
First of all - what is the slice? It is three things:
1. pointer to the underlying data,
2. length of the slice,
3. capacity of the slice.

These three things make a slice. If an empty slice means the slice with length of 0 then we can create such entity in
at least three different ways. 

### [1. Zero value slice](#zero-value-slice)
Zero value of a slice is the `nil`. It has capacity of 0, length of 0 and no underlying array.
```go
var stringSlice []string
``` 
### [2. Short declaration](#short-declaration)
Short declaration produces slice with capacity of 0, length of 0 and __pointer to the underlying data__.
```go
stringSlice := []string{}
```

### [3. Make](#make)
Make function similarly to short declaration produces slice with capacity of 0, length of 0
and __pointer to the underlying data__.
```go
stringSlice := make([]string, 0)
```
 
Each method produces slice for which condition `len(stringSlice) == 0` holds true. However, condition `stringSlice == nil`  is true only for zero value slice. It is in the spirit of Go for zero value to be useful. Indeed, zero value slice  is useful. Sometimes however distinction between `nil` slice and non-`nil` empty slice can be handy. Let's imagine a case where you want to collect information about professional experience from the user. Professional experience can be modelled as a slice of workplaces. You might want to distinguish three different situations.
1. user didn't provide professional experience yet,
2. user marked *I do not have any professional experience yet* checkbox,
3. user provided a list of his past professional endeavors.

Modelling the difference between cases 1. and 2. can be done with distinction of `nil` slice and non-`nil` empty slice. Luckily for us when marshaling into JSON - Go distinguishes between these cases.

```go
var stringSlice []string
json.Marshal(stringSlice) // Marshals into `null`
```

```go
stringSlice := []string{}
json.Marshal(stringSlice) // Marshals into `[]`
```

## [Empty slices in the DynamoDB](#empty-slices-dynamo)

What happens when we try to save an item into the DynamoDB when one of the attributes is empty slice?

There is no problem with zero value slice. You can save `nil` slice into DynamoDB and after retrieving it, it will be unmarshalled as a `nil` slice. A problem occurs when using empty non-`nil` slices. Look at the test.

```go
t.Run("regular way", func(t *testing.T) {
  t.Skip("this test fails")
  attrs, err := attributevalue.Marshal([]string{})
  assert.NoError(t, err)

  var s []string
  err = attributevalue.Unmarshal(attrs, &s)
  assert.NoError(t, err)

  assert.NotNil(t, s) // fails
  assert.Len(t, s, 0)
})
```

This failing test shows exactly what is the problem. I mean the problem isn't with the DynamoDB, but rather with how AWS SDK for Go treats slices. The DynamoDB is 100% capable of distinguishing between empty lists and `NULL` values. In order to see that let's look at `attrs`variable from the example above.
 
```go
(*dynamodb.AttributeValue)(0xc0000d5ea0)({
  NULL: true
})
```
This is what `attributevalue.Marshal` function did to non-`nil` empty slice. It changed into `NULL`. Actually as you can see `NULL` is a `bool` field on `dynamodb.AttributeValue` type. This is just how AttributeValue represents something that is `NULL`. Let's keep that in mind.

What we really want to do if we care about distinction between `nil` slice and non-`nil` empty slice is to use the custom encoder and decoder and set the `NullEmptySets` option that will preserve empty list.

```go
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
```

This test passes. Let's see how `attrs` variable looks like.
```go
(*dynamodb.AttributeValue)(0xc00024c140)({
  L: []
})
```
This is truly empty list. This is what we want! Do we have a success? Not yet...

The `attributevalue` package is used to transform application types into attribute values - which are types that the DynamoDB understands. It is used when performing for example `PutItem` operation or `GetItem` operation.

But we have also `UpdateItem` operations and typically we construct them with expression API. Let's look at the example.
```go
expr, err := expression.NewBuilder().
  WithUpdate(expression.Set(expression.Name("attr_name"), expression.Value([]string{}))).
  Build()
```  
This piece of code is an expresion that should update `attr_name` to the value of `[]string{}` which is empty non-`nil` slice. Let's print that expression and observe what are we dealing with.

```go
(expression.Expression) {
 expressionMap: (map[expression.expressionType]string) (len=1) {
  (expression.expressionType) (len=6) "update": (string) (len=12) "SET #0 = :0\n"
 },
 namesMap: (map[string]*string) (len=1) {
  (string) (len=2) "#0": (*string)(0xc000051990)((len=3) "123")
 },
 valuesMap: (map[string]*dynamodb.AttributeValue) (len=1) {
  (string) (len=2) ":0": (*dynamodb.AttributeValue)(0xc0002245a0)({
  NULL: true
})
 }
}
```

This is how update expression looks under the hood. We have `expressionMap`, `namesMap` and `valuesMap` parameters. Expression itself is `SET #0 = :0`. Path `#0` is substituted with `"123"` and `":0"` is substituted with `{NULL: true}`. We never really talked about how AWS SDK encodes expressions into the DynamoDB format because we have the expression API, and we didn't need to think about actual representation. At least until now. As you can see, even though we used `expression.Value([]string{})` it was transformed into the DynamoDB `NULL` value which isn't very good because we didn't want to have nil slice there. As a matter of fact expression API will always transform any slice with length of 0 into `NULL` value. Are we helpless though? No, we are not. Instead of simply using `expression.Value` we need to construct that  value by hand.

```go
v := expression.Value((&dynamodb.AttributeValue{}).SetL([]*dynamodb.AttributeValue{}))
expr, _ := expression.NewBuilder().WithUpdate(expression.Set(expression.Name("attr_name"), v)).Build()
```

This isn't very pretty, but it works. Now we get:
```go
(expression.Expression) {
 expressionMap: (map[expression.expressionType]string) (len=1) {
  (expression.expressionType) (len=6) "update": (string) (len=12) "SET #0 = :0\n"
 },
 namesMap: (map[string]*string) (len=1) {
  (string) (len=2) "#0": (*string)(0xc000051a20)((len=9) "attr_name")
 },
 valuesMap: (map[string]*dynamodb.AttributeValue) (len=1) {
  (string) (len=2) ":0": (*dynamodb.AttributeValue)(0xc000222780)({
  L: []
})
 }
}
```
The expression is aware of the empty list.

## [Summary](#summary)
Sometimes we need to get our hands dirty and tinker with the internals of the API in order to get things done. But from now on - we'll have complete confidence that we know what is going on with our slices when we use them with Dynamo.