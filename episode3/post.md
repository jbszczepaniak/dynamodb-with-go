# DynamoDB with Go #3
## Use this 4 tricks to build filesystem in DynamoDB under 15 minutes!
After reading previous episode you could have an impression that DynamoDB is just simple key-value store. I would like to straighten things out because DynamoDB is much more than that. Last time I also mentioned that keys can be more complicated. Get ready to dive into the topic of keys in DynamoDB with Go #3!

## [Let's build filesystem](#lets-build-filesystem)

Maybe it won't be full fledged filesystem, but I would like to model tree-like structure (width depth of 1, nested directories not allowed) where inside a directory there are many files. Moreover, I would like to query this filesystem in two ways:
1. Give me single file from given directory,
2. Give me whole directory.

These are my __access patterns__. I want to model my table in a way that will allow me to perform such queries.

## [Composite Primary Key](#composite-primary-key)

__Composite Primary Key__ consists of __Partition Key__ and __Sort Key__. Without going into details (AWS documentation covers this subject thoroughly), pair of Partition Key and Sort Key identifies an item in the DynamoDB. Many items can have the same Partition Key, but each of them needs to have different Sort Key. If you are looking for an item in the table and you already know what is the Partition Key then Sort Key narrows down your search to the specific item so to speak.

If table has defined only Partition Key - each item is recognized uniquely by its Partition Key. If however table is defined with Composite Primary Key each item is recognized by pair of Partition and Sort keys.

## [Table definition](#table-definition)

With all that theory in mind, let's figure out what should be Partition Key and what should be the Sort Key in our filesystem.

Each item in the table will represent single file. Additionally, each file must point to its parent directory. As I mentioned, Sort Key kind of narrows down the search. In this example, knowing already what directory we are looking for, we want to narrow down the search to single file.

All that suggests that `directory` should be the Partition Key and `filename` the Sort Key. Let's express it as CloudFormation template.

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
We need to define two attributes (`directory` and `filename`), because both of them are part of the __Composite Primary Key__. As you can see there is no sort key in the template. There is however `RANGE` key type. Just remember that:
- `HASH` key type corresponds to _f_Partition Key__
- `RANGE` key type corresponds to  __Sort Key__

## [Moving on to the code](#code)

This is how single item in the DynamoDB is going to look.
```go
type Item struct {
  Directory string `dynamodbav:"directory"`
  Filename  string `dynamodbav:"filename"`
  Size      string `dynamodbav:"size"`
}
```

I am going to insert couple of items to the database so that we have content we can query. At the end I want to have something like this in the table. (code that is doing that is omitted for brevity, you can look it up in `episode3/insert.go`)

| Directory | Filename       | Size |
| ---       | ----           | ---- |
| finances  | report2017.pdf | 1MB  |
| finances  | report2018.pdf | 1MB  |
| finances  | report2019.pdf | 1MB  |
| finances  | report2020.pdf | 2MB  |
| fun       | game1          | 4GB  |

## [Query #1: Give me single file from given directory](#query1)

We need to start with database setup.

```go
func TestSingleFileFromDirectory(t *testing.T) {
  ctx := context.Background()
  db, table := testingdynamo.SetupTable(t, "FileSystem")
  Insert(ctx, db, table)
```

With connection to DynamoDB in place and with testing data inserted we can move on to the query itself. I want to obtain single element from Dynamo, thus I am going to use `GetItemWithContext`.

```go
out, err := db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
  Key: map[string]*dynamodb.AttributeValue{
    "directory": dynamo.StringAttr("finances"),
    "filename":  dynamo.StringAttr("report2020.pdf"),
  },
  TableName: aws.String(table),
})
assert.NoError(t, err)
```

Note that `Key` consists of two elements: `directory` which is partition key, and `filename` - sort key. The same query can look a little bit differently.

```go
out, err := db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
  Key: map[string]*dynamodb.AttributeValue{
    "directory": {
      S: aws.String("finances"),
    },
    "filename":  {
      S: aws.String("report2020.pdf"),
    },
  },
  TableName: aws.String(table),
})
```

I learned very recently that I don't need to construct by hand `dynamodb.AttributeValue`. I highly recommend to use `dynamo` package from kuna's `platform` module because ability  to use `dynamo.StringAttr` is just so convenient! Just compare both code snippet and judge by yoursef!

Let's make sure that output of the query is really what we think it is:

```go
var item Item
err = dynamodbattribute.UnmarshalMap(out.Item, &item)
assert.NoError(t, err)
assert.Equal(t, Item{Directory: "finances", Filename: "report2020.pdf", Size: "2MB"}, item)
```

## [Query #2: Give me whole directory](#query2)

In this query we cannot really use `GetItemWithContext` because we want many items from Dynamo and to get single item we need to know whole composite primary key. Here we know only the partition key. Solution to that problem is `QueryWithContext` method with __Key Condition Expression__.
```go
expr, err := expression.NewBuilder().
    WithKeyCondition(
      expression.KeyEqual(expression.Key("directory"), expression.Value("finances"))).
    Build()
assert.NoError(t, err)

out, err := db.QueryWithContext(ctx, &dynamodb.QueryInput{
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),
  KeyConditionExpression:    expr.KeyCondition(),
  TableName:                 aws.String(table),
})
assert.NoError(t, err)
assert.Len(t, out.Items, 4)
```

That is a lot of new things, so let me break it down for you. First part is where we construct key condition expression which describes what we really want to query. In our case this is just _"Give me all items whose directory attribute is equal to `finances`"_. I am using expression builder which simplifies construction of expressions by far.

In the next step we are using expression inside the query. We need to provide __condition__, __names__, and __values__. In this example condition is just equality comparison, names correspond to names of attributes and values correspond to... their values!

Expression object gives us easy access to condition, names, and values. As you can see I am using them as parameters to `QueryInput`.

At the end, I am just checking whether we really have 4 items which are in finances directory.

## [Bonus - query #3 Give me reports before 2019](#query3)

It turns out that I constructed filenames in a way that makes them sortable. I figured - let's try to use it to our advantage and get all reports created before 2019.

Query stays exactly the same. Only thing we need to change is key condition expression.

```go
expr, err := expression.NewBuilder().
WithKeyCondition(
  expression.KeyAnd(
    expression.KeyEqual(expression.Key("directory"), expression.Value("finances")),
    expression.KeyLessThan(expression.Key("filename"), expression.Value("report2019")))).
Build()
assert.NoError(t, err)
```

We have 2 conditions that we combine with the AND clause. First one specifies what is our partition key, second one - sort key. `KeyLessThan` makes sure that we will only get `report2018.pdf` and `report2017.pdf`.

## [Summary](#summary)
I skipped few parts of the code so I am inviting you to clone this repository and play with it!

We learned today how to use composite primary keys. Moreover we know how to take advantage of them with Go!

We will use expression builder a lot in future episodes so get used to it. It takes some time to build intuition around using expression builder API, but it's totally worth it! 
