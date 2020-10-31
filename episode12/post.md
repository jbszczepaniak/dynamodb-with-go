# DynamoDB with Go #12

Item collection is a set of attributes that share the same Partition Key. A common pattern is to model one to many relationships with item collections where PK represents commonality among items and SK distinguishes types of items.

As an example we can take information about user and orders placed by the user.

| PK         | SK       |
| ---        | ----     |
| 1234       | USERINFO |
| 1234       | ORDER#2017-03-04 00:00:00 +0000 UTC|
| 1234       | ORDER#2017-02-04 00:00:00 +0000 UTC|

This is Single Table Design. We put different types of things into single table so that we can satisfy access patterns with minimal interactions with DynamoDB table. In this example we can display on a single page information about user with the latest orders while doing single query. PK represents ID of the user, and SK depending on the case represents basic user information or the order.

Now, that we've established that querying data gets simpler with Single Table Design, let us consider inserting data.

We need to be able to register a user. This use case is fairly simple. We need to make sure, that user with given ID doesn't exist in our system yet. The tool to do just that is to use expression like this one.

```go
expr, err := expression.NewBuilder().
  WithCondition(expression.AttributeNotExists(expression.Name("pk"))).
  Build()
```

When we have a user, we can place orders. Here is the thing. I would like to be able to place orders only for existing users. I would like to do something so that test passes. We want to get error when placing an order for the user that doesn't exist yet.

```go
func TestInsertingOrderFailsBecauseUserDoesNotExist(t *testing.T) {
  ctx := context.Background()
  tableName := "ATable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
  defer cleanup()

  _, err := db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
    Item: map[string]*dynamodb.AttributeValue{
      "pk": dynamo.StringAttr("1234"),
      "sk": dynamo.StringAttr("ORDER#2017-03-04 00:00:00 +0000 UTC"),
    },
    TableName: aws.String(tableName),
  })
  assert.Error(t, err)
}
``` 

Don't consider this a real world unit test. I am using `_test.go` file merely for executing queries. Anyway, this test fails. There is no error. Can we write simple condition that makes sure that user for which we want to place order exists?

Unfortunately, we can't do this _easily_. Writing conditions that relate to other items in the Item Collection is impossible. In order to make sure that user exists while introducing new order we need to use the transaction.

```go
func TestInsertingOrderFailsBecauseUserDoesNotExist(t *testing.T) {
  ctx := context.Background()
  tableName := "ATable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
  defer cleanup()

  expr, err := expression.NewBuilder().
    WithCondition(expression.AttributeExists(expression.Name("pk"))).
    WithUpdate(expression.Add(expression.Name("orders_count"), expression.Value(1))).
    Build()
  assert.NoError(t, err)

  _, err = db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
    TransactItems: []*dynamodb.TransactWriteItem{
      {
        Put: &dynamodb.Put{
          Item: map[string]*dynamodb.AttributeValue{
            "pk": {S: aws.String("1234")},
            "sk": {S: aws.String("ORDER#2017-03-04 00:00:00 +0000 UTC")},
          },
          TableName: aws.String(tableName),
        },
      },
      {
        Update: &dynamodb.Update{
          ConditionExpression:       expr.Condition(),
          ExpressionAttributeValues: expr.Values(),
          ExpressionAttributeNames:  expr.Names(),
          UpdateExpression:          expr.Update(),
          Key: map[string]*dynamodb.AttributeValue{
            "pk": {S: aws.String("1234")},
            "sk": {S: aws.String("USERINFO")},
          },
          TableName: aws.String(tableName),
        },
      },
    },
  })

  assert.Error(t, err)
  _, ok := err.(*dynamodb.TransactionCanceledException)
  assert.True(t, ok)
}
```

Transaction has two items. First one (Put) is what we really want to do here which is placing new order. Additionally, there is an update for item that represents information about the user. I did it so that I can put condition on that update operation. The condition says exactly what we wanted. Transaction will succeed only for existing user.

You can notice however that I am increasing `orders_count` which wasn't talked about as a requirement for this example. It doesn't really matter what gets updated, but you cannot perform empty update. We could choose to update some helper attribute to `""` but I wanted to make it useful. We are killing two birds with one stone. We get orders count for free.

Takeaway from this episode is the following one.
> If you want to base the condition of your operation on other item - you need to use transaction.
