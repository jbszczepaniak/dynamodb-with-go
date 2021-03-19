# DynamoDB with Go #2 - Put & Get
Today we are going to do the simplest thing you could imagine with the DynamoDB. First we are going to put something in, then we will take it out. It seems too easy and not worth reading about but bear with me for a moment.

In the first episode of the series we successfully created environment in which we are going to play with DynamoDB. You can find code for this episode
in [episode2](.) directory.

## [Database layout](#database-layout)

With the environment ready to go we can start solving problems. Our _problem_ for today is to save and retrieve basic information about Order. Order has three properties.
- `id` - string
- `price` - number
- `is_shipped` - boolean

In order to define the database layout I am using CloudFormation - assembly language for AWS infrastructure. You can
create DynamoDB tables via different channels for example using AWS CLI or with AWS console. I chose AWS CloudFormation
because if you work with serverless applications using Serverless Application Model (SAM) or Serverless framework - this
is how you are going to define your tables.
 
Let’s see how it looks.

```yaml
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  OrdersTable:
    Type: AWS::DynamoDB::Table
    Properties:
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      BillingMode: PAY_PER_REQUEST
      TableName: OrdersTable

```

First of all the table is called __OrdersTable__. Next, let’s focus on __AttributeDefinitions__. _Attributes_ in DynamoDB are fields or properties that you store inside an _item_. In the template we defined the `id` attribute of type string.

_Items_ are identified uniquely by their _keys_ - which are defined in the __KeySchema__ section. Key `id` is of type __HASH__ which is also referred to as the __Partition Key__. I won’t dive into types of keys at this time. For now, you need to know that Partition Key a.k.a. HASH Key uniquely identifies item in the DynamoDB table.

### Why `price` and `is_shipped` attributes aren’t defined?
In DynamoDB we need to define only the attributes that are part of the key. This is NoSQL world and we don’t need to specify each and every attribute of the item.

##  [Let’s see some code already!](#lets-see-code)
There you go. This will be our order definition. Notice `dynamodbav` struct tag which specifies how to serialize a given struct field. By the way __av__ in dynamodbav stands for attribute value.

```go
type Order struct {
  ID           string    `dynamodbav:"id"`
  Price        int       `dynamodbav:"price"`
  IsShipped    bool      `dynamodbav:"is_shipped"`
}
```
Let’s start with DynamoDB connection setup:

```go
func TestPutGet(t *testing.T) {
  ctx := context.Background()
  tableName := "OrdersTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
  defer cleanup()
```
Note that we defer calling `cleanup` function. This method removes the table that we created calling the `SetupTable`.

Now we need to prepare data before inserting it into DynamoDB.

```go
order := Order{ID: "12-34", Price: 22, IsShipped: false}
avs, err := attributevalue.MarshalMap(order)
assert.NoError(t, err)
```

Thanks to `dynamodbav` struct tags on the `Order`, `MarshalMap` function knows how to marshal struct into structure that DynamoDB understands. We are finally ready to insert something into DB.

```go
_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
	TableName: aws.String(tableName),
	Item:      avs,
})
assert.NoError(t, err)
```

We are using DynamoDB __PutItem__ operation which creates new item or replaces old item with the same key. First parameter is `context` which is used for cancellation. Second argument is `dynamodb.PutItemInput`. For every call to the AWS that SDK supports, you may expect this pattern:
- `APICall` - function to call
- `APICallInput` - argument for the function
- `APICallOutput` - return value of the function

One thing to notice is that `table` is wrapped in call to `aws.String` function. This is because in many places SDK accepts type `pointer to type` instead of just `type` and this wrapper makes that conversion.

Notice that first return value from the SDK call is being ignored. We don't really need it here. Only thing we want to know at this point is that we didn't get any errors.

## [Get order back from DynamoDB](#get-order-back)

```go
out, err := db.GetItem(ctx, &dynamodb.GetItemInput{
	Key: map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{
			Value: "12-34",
		},
	},
	TableName: aws.String(tableName),
})
assert.NoError(t, err)
```

Many pieces here are similar. There is `APICall`, and `APICallInput` elements that match pattern I showed you before. `TableName` parameter in the input is exactly the same.
Since we want to get item, we need to provide the key. This is where I find SDK cumbersome. Constructing keys looks a little bit off, but it is what it is. It is a map because key can be more complicated than what we have here. Remember how we defined `id` in the `template.yml`? It was of type "S" which is string. We need to specify that in the key as well when talking with the DynamoDB.

Last steps we need to perform are deserializing whatever we got from DynamoDB, and just to be sure - comparing results with what was put in.

```go
var queried Order
err = attributevalue.UnmarshalMap(out.Item, &queried)
assert.NoError(t, err)
assert.Equal(t, Order{ID: "12-34", Price: 22, IsShipped: false}, queried)
```

## [Summary](#summary)
Let me recap what we did today:

1. we defined database layout,
2. we marshaled a struct into DynamoDB item,
3. we put an item into DynamoDB,
4. we got item out of Dynamo,
5. we unmarshaled an item back into struct.


Make sure to clone the [repository](https://github.com/jbszczepaniak/dynamodb-with-go) and play with the code. Code related to this episode is in [episode2](.) directory in the repository.
