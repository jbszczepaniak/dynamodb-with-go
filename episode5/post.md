# DynamoDB with Go #5

Imagine that you are developing a brand new software that is based on information from the legacy system. The way you integrate with the legacy is through events that you are listening to. Let's say that you receive information about orders from an event. Each order is identifiable via unique id given by the legacy system that is an incrementally increased integer. One of the requirements for you is to process orders in a way that won't reveal total amount of orders.

It seems that the problem to solve is to map the old id to the new value. One thing to remember here is that we can receive many events about the same order which means that we need to either generate new id or use the id that we already generated for the legacy id. How can we do it with the DynamoDB?

## [Table definition](#table-definition)

```yaml
Resources:
  LegacyIDsTable:
    Type: AWS::DynamoDB::Table
    Properties:
      AttributeDefinitions:
        - AttributeName: old_id
          AttributeType: S
      KeySchema:
        - AttributeName: old_id
          KeyType: HASH
          Projection:
            ProjectionType: ALL
      BillingMode: PAY_PER_REQUEST
      TableName: LegacyIDsTable
```      

Very straightforward. Just a single attribute that is also partition key.

## [Failing test first](#failing-test-first)

Let's go here with the TDD approach. I will write a failing test first, then I will try to make it pass!

```go
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
    assert.NotEqual(t, "456", first)

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
```  

As you can see I have two requirements I want to cover. First of all I need to generate an id for each incoming legacy id that enters my function. Second of all, if I invoke that function twice with the same legacy id it shouldn't regenerate new id.

## [Structs setup](#structs-setup)

Let's take care of basic structs.

```go
type Mapper struct {
  db    dynamodbiface.DynamoDBAPI
  table string
}

type mapping struct {
  OldID string `dynamodbav:"old_id"`
  NewID string `dynamodbav:"new_id"`
}
```

There is `Mapper` that holds dependencies to the DynamoDB and the `mapping` that will be an item we store in the DynamoDB. Moreover external packages need to create the `Mapper`.

```go
func NewMapper(client dynamodbiface.DynamoDBAPI, table string) *Mapper {
  return &Mapper{db: client, table: table}
}
```

## [Solution](#solution)

Now let's think of the logic of the `Map` function. If the mapping of an old id and new id already exists in Dynamo - we need to fetch it. If it doesn't, we need to generate new id and save it. At first glance we could just use the `GetItemWithContext` and if we get nothing we just use thr `PutItemWithContext` to save newly generated id. Unfortunately this approach won't work. Remember that we can receive many events about the same order. They can arrive any time and can be handled concurrently. If two threads of execution will run the `GetItemWithContext` in more or less the same time and both will figure out that there is no mapping yet, we will end up with one of the thread overriding the other threads mapping.

Let's see how can we implement that functionality that will work in the world of concurrent execution.

```go
func (m *Mapper) Map(ctx context.Context, old string) (string, error) {
  idsMapping := mapping{OldID: old, NewID: uuid.New().String()}
  attrs, err := dynamodbattribute.MarshalMap(&idsMapping)
```

At the beginning we create a mapping that has an old id and that generates new id using UUIDv4. Next thing we'll do isn't retrieving an item from Dynamo. Instead we will revert the logic. I want translate into the code the following sentence: __Put into the DynamoDB a mapping, but only if it doesn't exist yet.__ In order to do that, we need to write condition expression.
```go
expr, err := expression.NewBuilder().
  WithCondition(expression.AttributeNotExists(expression.Name("old_id"))).
  Build()
```
This expression will make sure that the `PutItem` operation fails if an item with the same partition key as ours with attribute `old_item` already exists.

```go
_, err = m.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
  ConditionExpression:       expr.Condition(),
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),
  Item:                      attrs,
  TableName:                 aws.String(m.table),
})
```

We ignore output of the function, but care about error very much. If there is no error - then we know that there wasn't any mapping like ours before and we just return to the caller newly generated id.

```go
if err == nil {
  return idsMapping.NewID, nil
}
```

If however there is an error we need to check what type of error that is. If this error tells us that condition failed, it means that mapping already exists and we can do additional `GetItem` operation to retrieve it. If this is any other error, something went terribly wrong.

```go
aerr, ok := err.(awserr.Error)
if ok && aerr.Code() == dynamodbErrCodeConditionalCheckFailedException {
  return "", err
}
``` 
At this point we know that the `PutItem` operation failed because our conditional failed.

This means that someone before us already mapped the legacy id to new id and we can retrieve it.

```go
out, err := m.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
  Key: map[string]*dynamodb.AttributeValue{
    "old_id": {S: aws.String(old)},
  },
  TableName: aws.String(m.table),
})
if err != nil {
  return "", err
}
return aws.StringValue(out.Item["new_id"].S), nil
```

# [Summary](#summary)

We did it - we no longer rely on ids from legacy system and our solution is bulletproof! There is however one downside to it. Only for the first time for given legacy id we will communicate with DynamoDB once. Subsequent calls to `Map` function require two sequential calls to the DynamoDB. Next time we will try to figure out whether we can do something about it. As always I am inviting you to clone this repository and play with queries yourself!