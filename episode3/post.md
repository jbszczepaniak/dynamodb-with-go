# DynamoDB with Go #3
## Use this 4 tricks to build filesystem in DynamoDB under 15 minutes!
After reading previous episode you could have an impression that DynamoDB is just simple key-value store. I would like to straighten things out because DynamoDB is much more than that. Last time I also mentioned that keys can be more complicated. Get ready to dive into the topic of keys in DynamoDB with Go #3!

## [Let's build filesystem](#lets-build-filesystem)

Maybe it won't be full fledged filesystem, but I would like to model tree-like structure (width depth of 1, nested directories not allowed) where inside a directory there are many files. Moreover, I would like to query this filesystem in two ways:
1. Give me single file from given directory,
2. Give me all files from given directory.

These are my __access patterns__. I want to model my table in a way that will allow me to perform such queries.

## [Composite Primary Key](#composite-primary-key)

__Composite Primary Key__ consists of __Partition Key__ and __Sort Key__. Without going into details (AWS documentation covers this subject thoroughly), a pair of Partition Key and Sort Key identifies an item in the DynamoDB. Many items can have the same Partition Key, but each of them needs to have a different Sort Key. If you are looking for an item in the table and you already know what is the Partition Key, Sort Key narrows down the search to the specific item.

If a table is defined only by a Partition Key; each item is recognized uniquely by its Partition Key. If, however, a table is defined with a Composite Primary Key; each item is recognized by pair of Partition and Sort keys.

## [Table definition](#table-definition)

With all that theory in mind, let's figure out what should be a Partition Key and a Sort Key in our filesystem.

Each item in the table will represent a single file. Additionally, each file must point to its parent directory. As I mentioned, a Sort Key kind of narrows down the search. In this example, knowing already what directory we are looking for, we want to narrow down the search to a single file.

All that suggests that `directory` should be the Partition Key and `filename` the Sort Key. Let's express it as a CloudFormation template.

```yaml
Resources:
  FileSystemTable:
    Type: AWS::DynamoDB::Table
    Properties:
      AttributeDefinitions:
        - AttributeName: directory
          AttributeType: S
        - AttributeName: filename
          AttributeType: S
      KeySchema:
        - AttributeName: directory
          KeyType: HASH
        - AttributeName: filename
          KeyType: RANGE
      BillingMode: PAY_PER_REQUEST
      TableName: FileSystemTable
```
We need to define two attributes (`directory` and `filename`), because both of them are part of the __Composite Primary Key__. As you can see there is no Sort Key in the template. There is however `RANGE` key type. Just remember that:
- `HASH` key type corresponds to __Partition Key__
- `RANGE` key type corresponds to  __Sort Key__

## [Moving on to the code](#code)

This is how a single item in the DynamoDB is going to look.
```go
type item struct {
  Directory string `dynamodbav:"directory"`
  Filename  string `dynamodbav:"filename"`
  Size      string `dynamodbav:"size"`
}
```
I am going to insert a couple of items to the database so that we have content to query. Code that is doing that is omitted for brevity, you can look it up [here](./queries_test.go). At the end I want to have following table:

| Directory | Filename       | Size |
| ---       | ----           | ---- |
| finances  | report2017.pdf | 1MB  |
| finances  | report2018.pdf | 1MB  |
| finances  | report2019.pdf | 1MB  |
| finances  | report2020.pdf | 2MB  |
| fun       | game1          | 4GB  |

## [Query #1: Give me single file from given directory](#query1)
We need to start with a database setup.
```go
func TestSingleFileFromDirectory(t *testing.T) {
  ctx := context.Background()
  tableName := "FileSystemTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "./template.yml")
  defer cleanup()
  insert(ctx, db, tableName)
```
With a connection to DynamoDB in place and with the testing data inserted, we can move on to the query itself. I want to obtain a single element from the DynamoDB, thus I am going to use `GetItemWithContext`.
```go
out, err := db.GetItem(ctx, &dynamodb.GetItemInput{
	Key: map[string]types.AttributeValue{
		"directory": &types.AttributeValueMemberS{Value: "finances"},
		"filename":  &types.AttributeValueMemberS{Value: "report2020.pdf"},
	},
  TableName: aws.String(tableName),
})
```
Note that `Key` consists of two elements: `directory` (Partition Key) and `filename` (Sort Key). Let's make sure that output of the query is really what we think it is:
```go
var i item
err = attributevalue.UnmarshalMap(out.Item, &i)
assert.NoError(t, err)
assert.Equal(t, item{Directory: "finances", Filename: "report2020.pdf", Size: "2MB"}, i)
```
## [Query #2: Give me whole directory](#query2)

In this query we cannot use `GetItem` because we want to obtain many items from the DynamoDB. Also when we get a single item we need to know whole composite primary key. Here we want to get all the files from the directory so we know only the Partition Key. Solution to that problem is `Query` method with __Key Condition Expression__.
```go
expr, err := expression.NewBuilder().
  WithKeyCondition(
    expression.KeyEqual(expression.Key("directory"), expression.Value("finances"))).
  Build()
assert.NoError(t, err)

out, err := db.Query(ctx, &dynamodb.QueryInput{
	ExpressionAttributeNames:  expr.Names(),
	ExpressionAttributeValues: expr.Values(),
	KeyConditionExpression:    expr.KeyCondition(),
	TableName:                 aws.String(tableName),
})
assert.NoError(t, err)
assert.Len(t, out.Items, 4)
```

That is a lot of new things, so let me break it down for you. First part is where we construct key condition expression which describes what we really want to query. In our case this is just _"Give me all items whose directory attribute is equal to `finances`"_. I am using an expression builder which simplifies construction of expressions by far.

In the next step we are using expression inside the query. We need to provide __condition__, __names__, and __values__. In this example, condition is just an equality comparison, where names correspond to names of attributes and values correspond to... their values!

An expression object gives us easy access to condition, names, and values. As you can see I am using them as parameters to `QueryInput`.

At the end, I am just checking whether we really have 4 items which are in finances directory.

## [Bonus - Query #3 Give me reports before 2019](#query3)

It turns out that I constructed filenames in a way that makes them sortable. I figured - let's try to use it to our advantage and get all the reports created before 2019.

Query stays exactly the same. The only thing we need to change is the key condition expression.

```go
expr, err := expression.NewBuilder().
WithKeyCondition(
  expression.KeyAnd(
    expression.KeyEqual(expression.Key("directory"), expression.Value("finances")),
    expression.KeyLessThan(expression.Key("filename"), expression.Value("report2019")))).
Build()
assert.NoError(t, err)
```

We have two conditions that we combine with the AND clause. The first one specifies what is our Partition Key, second one, the Sort Key. `KeyLessThan` makes sure that we will only get `report2018.pdf` and `report2017.pdf`. Let's have a look at the results of the query.

```go
var items []item
err = attributevalue.UnmarshalListOfMaps(out.Items, &items)
assert.NoError(t, err)
if assert.Len(t, items, 2) {
  assert.Equal(t, "report2017.pdf", items[0].Filename)
  assert.Equal(t, "report2018.pdf", items[1].Filename)
}
```

In the first query we used `attributevalue.UnmarshalMap` for unmarshaling single DynamoDB item into the struct. We knew we will get single item. Here we know that there will be one item or more - thus we use `attributevalue.UnmarshalListOfMaps` - which unmarshals the query results into the slice of items.

Note that I assert that first item is the report from 2017 and second one is from 2018. How am I so sure that items will go back from the DynamoDB in that order? If not told otherwise - DynamoDB will scan items from given Partition in ascending order. Since 2017 comes before 2018 - I know that first item should be from 2017.

## [Summary](#summary)
We learned today how to use composite primary keys. Moreover we know how to take advantage of them with Go! That's great! You know what is even better? Playing with the code! Make sure to clone this repository and tinker with it!

Also we used expression builder to create the DynamoDB expression. Get used to them - we will use them a lot in the future episode! It takes some time to build intuition around using expression builder API, but it's totally worth it!
