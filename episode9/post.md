# DynamoDB with Go #9

Here is the scenario for this episode. There is a toggle, and it can be switched on and off. Whenever toggle is switched, an event is published and we are in charge of consuming that event. Our task is to save switches of state of the toggle and to be able to tell whether it is on or off at the moment. Bad news is that events can arrive out of order - an old event can appear at any time, and we need to be able to reject it.

It may seem like fictitious example but is 100% real. I dealt with this kind of problem couple of weeks ago. Let me show you how it can be done with the DynamoDB!

We really have simple access pattern here. We want to obtain the latest state of the toggle. Additionally, we should be able to obtain log of switches that happened. Let's jump into the tests that define what we really want to do.

## [Test suite](#test-suite)

```go
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
```

Can it be simpler? I don't think so - save it first - retrieve later. We want more however, next test proves that we can save many switches, and we can retrieve the latest one.

```go
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
```

Last test shows that out of order events are not taken into account. If an old event arrives, it doesn't influence the latest state.

```go
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
```

This is all we want from `the system`. In the next section I am going to provide you with the solution for this problem that I came up with. If you are able to provide more elegant solution, more efficient one or maybe just ...better, feel free to share it with me! I am keen to look at the problem at a different angle!

## [A solution](#a-solution)

Let's recap. Whenever new `Switch` is consumed by our function I want to append it at the end of my "log" but only if it is the newest one.

We do not have any table yet! We need to fix that. The table will have Partition Key called `pk`, and Sort Key called `sk`. As you saw in test cases the toggle has ID. Let's use that ID as the PK for every item connected with that toggle. In terms of SK - I want to use it twofold. First of all we will have log items. Each log item will have SK that starts with `READ` prefix followed by time of given switch. There will be additional special item with SK with constant "LATEST_SWITCH". This item will be used both to keep the order when writing a log item and when retrieving the latest switch.

## [Coding time](#coding-time)

```go
func (t *Toggle) Save(ctx context.Context, s Switch) error {
  item := s.asItem()
```

We'll start with keeping details unexported. `Switch` is a public thing, let's not pollute it with implementation details. The DynamoDB on the other hand needs to know how to deal with switches. We can also have different contents of `sk`. Because of that we have `asItem()` method.

```go
func (s Switch) asLogItem() switchItem {
  return switchItem{
    PK:        s.ID,
    SK:        "SWITCH#" + s.CreatedAt.Format(time.RFC3339Nano),
    CreatedAt: s.CreatedAt,
    State:     s.State,
  }
}
```

What is `switchItem`? I am glad you asked. It knows how to marshal/unmarshal items.

```go
type switchItem struct {
  PK string `dynamodbav:"pk"`
  SK string `dynamodbav:"sk"`

  State     bool      `dynamodbav:"state"`
  CreatedAt time.Time `dynamodbav:"created_at"`
}
```

Since we are speaking about marshaling we need to convert our data into the format that the DynamoDB understands.

```go
attrs, err := dynamodbattribute.MarshalMap(item)
``` 

Next thing is the most complicated expression we've ever seen in the DynamoDB with Go because it combines condition and update.

```go
expr, err := expression.NewBuilder().
  WithCondition(expression.LessThan(expression.Name("created_at"), expression.Value(item.CreatedAt))).
  WithUpdate(expression.
    Set(expression.Name("created_at"), expression.Value(item.CreatedAt)).
    Set(expression.Name("state"), expression.Value(item.State))).
  Build()
```

What is says is 
> Please update `created_at` field and `state` field but only if item we are inserting is younger than what is in the DynamoDB.

```go
_, err = t.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
  TransactItems: []*dynamodb.TransactWriteItem{
    {
      Update: &dynamodb.Update{
        Key: map[string]*dynamodb.AttributeValue{
            "pk": {S: aws.String(item.PK)},
            "sk": {S: aws.String("LATEST_SWITCH")},
        },
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
        ConditionExpression:       expr.Condition(),
        TableName:                 aws.String(t.table),
        UpdateExpression:          expr.Update(),
        ReturnValuesOnConditionCheckFailure: aws.String("ALL_OLD"),
      },
    },
    {
      Put: &dynamodb.Put{
        Item:      attrs,
        TableName: aws.String(t.table),
      },
    },
  },
})
```

We are using transaction because we want to append log entry (Put) and update the latest state of the toggle (Update) but only if the condition holds true for the latest state.

Let's put in english the condition mixed with this transaction. This is what is says:
> I want to update the `created_at` and the `state` fields on the item that represents the latest state of the switch, but only if what I am  trying to put into the DynamoDB is younger than what is already there. Additionally, I want to append the log item (because I want to have history of what happened to that toggle).

One more thing - `ReturnValuesOnConditionCheckFailure` parameter. It is crucial because it allows us to recognize what happened after transaction failure. It can fail for two reasons:
1. condition failed - we received out of order event - we need to discard it
2. database doesn't have any item with given `pk` - it is the first write for given partition key

```go
if err == nil {
  return nil
}
```

We might not have an error at all - that means that we've just appended new log entry with an update to the latest state.

```go
aerr, ok := err.(*dynamodb.TransactionCanceledException)
if !ok {
  return err
}
```

We might have an error that wasn't anticipated. In that case - application blows up, and we have an incident to handle.

```go
if len(aerr.CancellationReasons[0].Item) > 0 {
  return nil
}
```

Based on that condition we can reason that we received an out of order event. Why? Because we filled `ReturnValuesOnConditionCheckFailure` parameter with `ALL_OLD` value. This means that when transaction fails the value  of `aerr.CancellationReasons[0].Item` will be an item that was in the DynamoDB before our action. If that value is not empty this means that there is an item in the DynamoDB for given Partition Key. Since we have two reasons for transaction failure we can - by elimination - conclude that we'ce received an out of order event.
 
Out of order event isn't saved into the DynamoDB so we exit immediately. Now we need to handle the situation when transaction failed because it's the first time DynamoDB sees such Partition Key.

```go
expr, err = expression.NewBuilder().
   WithCondition(
     expression.Not(expression.And(
       expression.Equal(expression.Name("pk"), expression.Value(item.PK)),
       expression.Equal(expression.Name("sk"), expression.Value("LATEST_SWITCH")),
     ))).Build()
``` 

This condition is responsible for making sure that the DynamoDB didn't save `LATEST_SWITCH` for our Partition Key yet. It's a possibility because between failed transaction and now there is time difference. Some other process could have saved such item in between.

We also need to create an item representing the latest state of the toggle.

```go
latestAttrs, err := dynamodbattribute.MarshalMap(s.asLatestItem())
```
It's similar to `asLogItem` but it sets `sk` to LATEST_SWITCH`. Next thing we do is yet another transaction.
```go
_, err = t.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
  TransactItems: []*dynamodb.TransactWriteItem{
    {
      Put: &dynamodb.Put{
        ConditionExpression:       expr.Condition(),
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
        Item:                      latestAttrs,
        TableName:                 aws.String(t.table),
      },
    },
    {
      Put: &dynamodb.Put{
        Item:      attrs,
        TableName: aws.String(t.table),
      },
    },
  },
})
```
What we want to achieve here is appending first log item and create the latest state of the toggle which holds the same information as the only log item we've saved.

What can happen now?
```go
if err == nil {
  return nil
}
```
We might have no error - condition passed and both items were saved. This DynamoDB call can fail as well.
```go
_, ok = err.(*dynamodb.TransactionCanceledException)
if !ok {
  return err
}
```
If it does - and the reason isn't transaction failure - something wrong happened, and we return with an error. If however transaction failed - first switch for the toggle was saved but not by us. What we can do is to call this whole function again. It is completely save and won't create an infinite loop because the `Switch` is either older or younger than what is saved in the Dynamo. If it is younger it will be saved. If it's older - it will be rejected.

```go
return t.Save(ctx, s)
```

Very often - when writing is complex - reading must be trivial. This is the case in here!
```go
func (t *Toggle) Latest(ctx context.Context, userID string) (Switch, error) {
out, err := t.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
  Key: map[string]*dynamodb.AttributeValue{
      "pk": {S: aws.String(userID)},
      "sk": {S: aws.String("LATEST_SWITCH")},
  },
  TableName: aws.String(t.table),
})
if err != nil {
  return Switch{}, err
}
if len(out.Item) == 0 {
  return Switch{}, errors.New("not found")
}

var item switchItem
err = dynamodbattribute.UnmarshalMap(out.Item, &item)
return item.asSwitch(), err
}
```

We retrieve the latest state of the toggle, unmarshal it and return it to the client! 

## Summary
Fairly easy test suite received very complex implementation. If you know how to satisfy the same test suite with simpler implementation - let me know - I would love to know how! Nevertheless, we learned two important pieces of DynamoDB API. First of all we know that expression API can combine both updates and conditions. Other than that - now we know that after transaction fails - we can use `ReturnValuesOnConditionCheckFailure` to obtain more insight on what really happened.