# DynamoDB with Go #11

I am advocating for using the expression API from the first episode of the DynamoDB with Go. In this episode I would like to show you why I enjoy using them so much. We are going to examine three examples. Each of them is going to be implemented using plain text expressions and with expressions API. I hope that after examining these three comparisons you'll be convinced that expression API is the way to go.

## [Example #1 - Get item collection ](#example1)
When table uses Composite Primary Key (meaning that it has both PK and SK), an item collection is bunch of items that share the same Partition Key. In order to fetch the item collection we need to use DynamoDB Query with Key Condition. Let's start with a test.

```go
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
```

I am using `insert` helper that sets up the stage for me. We are going to begin each test with this one.

> I would like to draw your attention to the fact that `GetItemCollection` function has `V1` prefix. We also have version 2 of the function. For this episode I'll set a convention where all functions that use traditional string based expressions end with `V1` and are located in [strings_expressions.go](./strings_expressions.go) file. Functions that use expression API can be found in [api_expressions.go](./api_expressions.go) and end with `V2` prefix.

Let's compare implementations.

```go
out, err := db.QueryWithContext(ctx, &dynamodb.QueryInput{
  KeyConditionExpression: aws.String("#key = :value"),
  ExpressionAttributeNames: map[string]*string{
    "#key": aws.String("pk"),
  },
  ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
  	":value": {S: aws.String(pk)},
  },
  TableName: aws.String(table),
})
```

vs

```go
expr, err := expression.NewBuilder().
  WithKeyCondition(expression.KeyEqual(expression.Key("pk"), expression.Value(pk))).
  Build()

out, err := db.QueryWithContext(ctx, &dynamodb.QueryInput{
  KeyConditionExpression:    expr.KeyCondition(),
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),
  TableName:                 aws.String(table),
})
```

Certainly second version hides from us details that we need to think about when constructing expressions ourselves.

What we really want to do in this example is to construct condition that checks equality. Attribute `"pk"` should be equal to the variable `pk`. Doing it directly is risky because we cannot use reserved keywords of the DynamoDB. This is why we create aliases `#key` and `:value` which are later mapped to real values, `#key` becomes `"pk"`, and `:value` becomes contents of variable `pk`. When using expressions API, we don't need to build these by hand, the API is doing it automatically. Let's examine the expression that is built by the expression API.

```go
(expression.Expression) {
 expressionMap: (map[expression.expressionType]string) (len=1) {
  (expression.expressionType) (len=12) "keyCondition": (string) (len=7) "#0 = :0"
 },
 namesMap: (map[string]*string) (len=1) {
  (string) (len=2) "#0": (*string)(0xc0002b63b0)((len=2) "pk")
 },
 valuesMap: (map[string]*dynamodb.AttributeValue) (len=1) {
  (string) (len=2) ":0": (*dynamodb.AttributeValue)(0xc0002768c0)({
  S: "1"
})
 }
}
```

It is really similar. The only difference is that expression API creates aliases meaningless for humans like `#0` and `:0`. Note that alias for key has to start with `:` and alias for value with `#`. 

> Further parts of both versions of implementations are identical - so I am skipping them here. You can examine them in the repository. The same goes for second and third example. I am going to show only expressions - as they are the only parts that differ.

## [Example #2 - "update A and unset B but only if B is set to `baz`"](#example2)

Taken out of context doesn't really make sens, but this scenario is inspired by real life case. For given item that has attributes `A` and `B` I want to update `A` to some value and unset `B` but only if `B` is set to `baz`.

Let's jump into the test.

```go
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
```
It turns out that `item1` has `B` set to `bar` thus it's not updated, `item2` on the other hand has `B` set to `baz` and it is updated.

Let's compare implementations.

```go
out, err := db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
  ConditionExpression: aws.String("#b = :b"),
  ExpressionAttributeNames: map[string]*string{
    "#b": aws.String("b"),
    "#a": aws.String("a"),
  },
  ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
    ":b": {S: aws.String(whenB)},
    ":a": {S: aws.String(newA)},
  },
  Key:              marshaledKey,
  ReturnValues:     aws.String("ALL_NEW"),
  TableName:        aws.String(table),
  UpdateExpression: aws.String("REMOVE #b SET #a = :a"),
})
```

vs.

```go
expr, err := expression.NewBuilder().
  WithCondition(expression.Equal(expression.Name("b"), expression.Value(whenB))).
  WithUpdate(expression.
    Set(expression.Name("a"), expression.Value(newA)).
    Remove(expression.Name("b"))).
  Build()

out, err := db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
  ConditionExpression:       expr.Condition(),
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),
  UpdateExpression:          expr.Update(),
  Key:                       marshaledKey,
  ReturnValues:              aws.String("ALL_NEW"),
  TableName:                 aws.String(table),
})
```

I think that second approach is better because it is less error prone and gives more clarity (which is 100% subjective by the way). This example is fairly simple. The more fields you want to update, and more complex your conditions are - the more you'll appreciate type systems of Golang when constructing expressions with the expressions API. 

On a side note, notice how expression builder allows you to mix both condition expressions and update expressions.

## [Example #3 - Put if doesn't exist](#example3)

By default `PutItem` operation overwrites an item if it already exists. It is very common to make sure to not overwrite that, and the tool to do that is the condition expression. Here is the test.

```go
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
```

Item with PK=1 and SK=2 was already inserted to the DynamoDB, thus cannot be inserted again and test fails. For PK=10 and  SK=20 operation succeeds. Let's compare implementations.

```go
_, err = db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
  ConditionExpression: aws.String("attribute_not_exists(#pk)"),
  ExpressionAttributeNames: map[string]*string{
  	"#pk": aws.String("pk"),
   },
   Item:      marshaledKey,
   TableName: aws.String(table),
 })
```
 
vs.

```go
expr, err := expression.NewBuilder().
  WithCondition(expression.AttributeNotExists(expression.Name("pk"))).
  Build()

_, err = db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
  ConditionExpression:      expr.Condition(),
  ExpressionAttributeNames: expr.Names(),
  Item:                     marshaledKey,
  TableName:                aws.String(table),
})
```

If I am being honest - I kind of like first version. It is concise, you immediately know what is going. The thing is that I think that conventions matter, and I'd rather stick with one way of doing things. Other thing is that even though `attribute_not_exists(#pk)` is cute - it is so simple to make mistake - and you don't have any autocompletion when writing it by hand.

## [Summary](#summary)
I think that ability to write expressions by hand matters. I believe that this ability helps along the way when you're trying to figure out when your query breaks. Having said that - when you know what is what - I think in day to day work it is better to stick with expression API as it very convenient and less error prone than plain text expressions.
