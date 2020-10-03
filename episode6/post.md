# DynamoDB with Go #6

I finished last episode with a promise to show you how to approach what we discussed differently. If you want to get most out of this episode I highly recommend you to read [episode #5](../episode5/post.md) first. In a nutshell a problem to solve was to map ids from existing legacy system into new system that will use its own ids schema.

## [Previous approach](#previous-approach)

Initial solution for this problem was to generate the new id for the legacy id and conditionally put the mapping into DB. The condition stated that we don't want to do that if mapping for given legacy id already exists. When condition succeeded we  knew that there is no mapping for given legacy id yet and mapping was inserted into the DynamoDB. If however condition failed we knew that mapping already exists, and we need to grab it from the database.

This approach works very well and is immune to data races. When we call `Map` function for the first time for given legacy id, we need to make single call to the DynamoDB. The condition succeeds, and we return newly generated id. However, subsequent calls to the `Map` with the same id will require two calls to the database. First failing call because of the condition we have in place, and second one that grabs already existing new id from the Dynamo.

## [ReturnValues parameter](#return-values-parameter)

It turns out that there is a parameter in the `PutItem` called `ReturnValues`. It can be set to `NONE` (default) or `ALL_OLD`. `ALL_OLD` means that if `PutItem` has overridden an item, then the response from the DynamoDB will contain content of the item before overriding it. This would be great for us. We would like to know - if we failed - what was in the DynamoDB that caused the failure. That would mean that we don't need to call Dynamo for the second time.

Unfortunately - this works only if `PutItem` succeeds and in case of conditional check failure we don't get an image of an item before the `PutItem` that failed.

Since that didn't work - maybe error that we get back that informs us about condition failure, contains more context on why it failed?

Unfortunately it doesn't.

We can however use this idea somewhere else to get just what we want.

## [Transactions](#transactions)

In general, transactions are for something else. You use them when you need to make many changes to the database state and all of them need to be successful in order for transaction to complete. If one of the changes in the transaction fails, then whole transaction fails and none of the changes are being made.

If the transaction fails then the DynamoDB can give you more context of what happened. Let's see how it works.

## [Code](#code)

All the code is the same as in the 5th episode. Only thing that changes is `Map` function. You can observe here also beauty of automated tests that verify only behaviour. We can change implementation as much as we want and tests do not need to change at all. Moreover, they'll tell us whether new approach works or not.

Let's change `Map` function to use the transaction.

```go
idsMapping := mapping{OldID: old, NewID: uuid.New().String()}
attrs, err := dynamodbattribute.MarshalMap(&idsMapping)
if err != nil {
  return "", err
}
expr, err := expression.NewBuilder().
  WithCondition(expression.AttributeNotExists(expression.Name("old_id"))).
  Build()
if err != nil {
  return "", err
}
```
When I told you that only `Map` function changes I didn't mean all of it. The beginning stays exactly the same. Just to recap, we create the mapping with the old id and the new id that gets generated for us. Then we construct the condition that fails when we want to put an item that already exists.

```go
_, err = m.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
  TransactItems: []*dynamodb.TransactWriteItem{
    {
      Put: &dynamodb.Put{
        ConditionExpression:                 expr.Condition(),
        ExpressionAttributeNames:            expr.Names(),
        ExpressionAttributeValues:           expr.Values(),
        Item:                                attrs,
        ReturnValuesOnConditionCheckFailure: aws.String("ALL_OLD"),
        TableName:                           aws.String(m.table),
      },
    },
  },
})
```
We have a transaction here with single `Put` operation in it. There is also something that is not an option for regular `PutItem` method - `ReturnValuesOnConditionCheckFailure` parameter. Hopefully when condition fails, we will get exactly what we want - which is the new id that already exists.

The key aspect here is error handling.
```go
if err == nil {
  return idsMapping.NewID, nil
}
``` 
If there is no error it means that condition succeeded, hence this was first time anyone called `Map` with given legacy id, and we just return what we've put into the DynamoDB.

```go
aerr, ok := err.(*dynamodb.TransactionCanceledException)
if !ok {
  return "", err
}
```
If there is an error we need to check its type, and if it is not `TransactionCanceledException` - something went wrong, and we don't know what it is, so we just return.

```go
return aws.StringValue(aerr.CancellationReasons[0].Item["new_id"].S), nil
```
Otherwise, we get `new_id` from `CancellationReasons` and we can return that to the client without calling Dynamo again!

## [Summary](#summary)

We just showed how can we leverage DynamoDB API to give us reason for transaction failure. In our case this means that we can insert into the DynamoDB new mapping or get existing mapping in one step.

Should you use this approach? Is it better than the previous one? As always it depends. When calling the DynamoDB only once you'll save time spent on request/response round trip. Does it come for free then? Absolutely not. Transaction calls to the DynamoFB are more expensive in terms of Capacity Units you'll pay for them. So in this particular case either you pay more for the DynamoDB transaction call, but make fewer calls in general, or pay less for single call but call DynamoDB more times and spend more time waiting for network calls to the DynamoDB. Having said that I am not recommending any of the approaches. It depends on your needs. I am just showing possible options.