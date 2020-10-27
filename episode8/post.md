# DynamoDB with Go #8

This episode is all about implementation. If you didn't read episode 7th - please [do](../episode7/post.md) because we can't move forward without it. Assuming you've already read it, let's express all the use cases we have with help of unit tests.

## [Registering a sensor](#registering-a-sensor)

Registering a sensor gives ability to record new readings of the sensor later on.

```go
t.Run("register sensor, get sensor", func(t *testing.T) {
  tableName := "SensorsTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
  defer cleanup()
  manager := sensors.NewManager(db, tableName)

  err := manager.Register(ctx, sensor)
  assert.NoError(t, err)

  returned, err := manager.Get(ctx, "sensor-1")
  assert.NoError(t, err)
  assert.Equal(t, sensor, returned)
})
``` 

In order to verify that registration went well - sensor is retrieved afterwards. You might wonder - isn't unit testing about testing single units? Isn't the registration __a unit__? Well, registration doesn't really matter if we cannot do anything with it, and a unit is single behavior. At the beginning of the journey of testing my code it was a little strange for me. I was thinking - "but now such test has 2 reasons to break". Thinking in terms of above example - when registration fails and when retrieval fails. Later on, I came to the conclusion that there is superiority of such test over - for example checking what method was called underneath because it is better to test behavior than implementation. This is the key aspect of testing for me because change in implementation shouldn't break tests when behavior doesn't change. Now let's go back to the sensor business. Next behavior we want to cover is inability to register a sensor with the same ID twice.

## [Inability to register the same sensor twice](#register-only-once)

```go
t.Run("do not allow to register many times", func(t *testing.T) {
  tableName := "SensorsTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
  defer cleanup()
  manager := sensors.NewManager(db, tableName)

  err := manager.Register(ctx, sensor)
  assert.NoError(t, err)

  err = manager.Register(ctx, sensor)
  assert.EqualError(t, err, "already registered")
})
```

When you try to register again you get slapped with an error. One thing I want to mention is `sensor` variable that I used in both above mentioned snippets. It's just exemplary sensor I've declared on top of the test suite. You can look it up in the repository if you want to.

## [Recording a sensor reading](#recording-sensor-reading)

Since we know how to register a sensor, let's record a reading of that sensor.

```go
t.Run("save new reading", func(t *testing.T) {
  tableName := "SensorsTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
  defer cleanup()
  manager := sensors.NewManager(db, tableName)

  err := manager.Register(ctx, sensor)
  assert.NoError(t, err)

  err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.67", ReadAt: time.Now()})
  assert.NoError(t, err)

  _, latest, err := manager.LatestReadings(ctx, "sensor-1", 1)
  assert.NoError(t, err)
  assert.Equal(t, "0.67", latest[0].Value)
})
```

After saving new reading I need to verify somehow that it worked. In order to do that I am using `LatestReadings` method that provides me with one latest reading - which hopefully should be the one that was just saved.

## [Retrieving sensor and its last readings](#retrieve-sensor-with-readings)

Let's explore more the API that we already've seen in previous test.

```go
t.Run("get last readings and sensor", func(t *testing.T) {
  tableName := "SensorsTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
  defer cleanup()
  manager := sensors.NewManager(db, tableName)

  err := manager.Register(ctx, sensor)

  assert.NoError(t, err)

  err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.3", ReadAt: time.Now().Add(-20 * time.Second)})
  assert.NoError(t, err)
  err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.5", ReadAt: time.Now().Add(-10 * time.Second)})
  assert.NoError(t, err)
  err = manager.SaveReading(ctx, sensors.Reading{SensorID: "sensor-1", Value: "0.67", ReadAt: time.Now()})
  assert.NoError(t, err)

  sensor, latest, err := manager.LatestReadings(ctx, "sensor-1", 2)
  assert.NoError(t, err)
  assert.Len(t, latest, 2)
  assert.Equal(t, "0.67", latest[0].Value)
  assert.Equal(t, "0.5", latest[1].Value)
  assert.Equal(t, "sensor-1", sensor.ID)
})
```  

The point of this test is to show that we are able to fetch a sensor and latest readings of this sensor at the same time.

## [Get sensors by location](#get-sensors-by-location)

```go
t.Run("get by sensors by location", func(t *testing.T) {
  tableName := "SensorsTable"
  db, cleanup := dynamo.SetupTable(t, ctx, tableName, "../template.yml")
  defer cleanup()
  manager := sensors.NewManager(db, tableName)

  err := manager.Register(ctx, sensors.Sensor{ID: "sensor-1", City: "Poznan", Building: "A", Floor: "1", Room: "2"})
  err = manager.Register(ctx, sensors.Sensor{ID: "sensor-2", City: "Poznan", Building: "A", Floor: "2", Room: "4"})
  err = manager.Register(ctx, sensors.Sensor{ID: "sensor-3", City: "Poznan", Building: "A", Floor: "2", Room: "5"})

  ids, err := manager.GetSensors(ctx, sensors.Location{City: "Poznan", Building: "A", Floor: "2"})
  assert.NoError(t, err)
  assert.Len(t, ids, 2)
  assert.Contains(t, ids, "sensor-2")
  assert.Contains(t, ids, "sensor-3")
})
```

This test is the reason why there are two versions of the code in this episode ([version 1](./v1) and [version 2](./v2)). Both versions have the same test suite we've just described but each of them has different implementation. They differ because I  wanted yo show you different approaches of handling hierarchical data modeling. 

Before we jump into implementations I wanted to let you know that I am well aware of the fact that the test suite  is not complete. There are corner cases that aren't covered, but I hope you'll realize that this is rather the DynamoDB modeling exercise rather than testing exercise. One more thing about testing. Both versions of the code have the same test suite because we are testing behavior - not implementation. I'm repeating myself, but I would like to emphasize the importance of such tests. I know that some times people are discouraged to test their code because whenever they change production code they need to fix a lot of tests too. It's not the case when testing the behavior. That's it. I used the word "behavior" for the last time in this episode. Moving on.

## [Implementation - version 1](#implementation-v1)

I am going to show you first version of implementation and then we are going to jump into the second one and 
compare them. Let me remind you what really is the first version.

| PK         | SK                    | Value | City   | Building  |  Floor |  Room  | Type | ID  |
| ---        | ----                  | ---   | ---    | ---       | ---    | ---    | ---  | --- |
| SENSOR#S1  | READ#2020-03-01-12:30 |   2   |        |           |        |        |      |     |
| SENSOR#S1  | READ#2020-03-01-12:31 |   3   |        |           |        |        |      |     |
| SENSOR#S1  | READ#2020-03-01-12:32 |   5   |        |           |        |        |      |     |
| SENSOR#S1  | READ#2020-03-01-12:33 |   3   |        |           |        |        |      |     |
| SENSOR#S1  | SENSORINFO            |       | Poznań |  A        |   2    |  13    | Gas  |     |
| CITY#Poznań| LOCATION#A#2#13       |       |        |           |        |        |      | S1  |

Information about a sensor is broken down into two separate items with different Partition Keys. Additionally, every recording is an item that shares the same PK as main item describing the sensor.

### [Registration](#registration)

As you can see in the layout, when registering we need to write two different items which means that we are going to need transactions. Let's jump into it.

```go
func (s *sensorManager) Register(ctx context.Context, sensor Sensor) error {
  attrs, err := dynamodbattribute.MarshalMap(sensor.asItem())
```

What we are doing here is transforming a sensor into something that we can put into the DynamoDB. Before we go any further, let's talk about `Sensor` type and `asItem` method. I differentiate here two different types: `Sensor` which is the public  representation of a sensor and additional type `sensorItem` that is concerned only with how sensor is stored in the DynamoDB. This type is unexported because it is only the implementation detail.

```go
type Sensor struct {
  ID       string
  City     string
  Building string
  Floor    string
  Room     string
}

type sensorItem struct {
  ID string `dynamodbav:"pk"`
  SK string `dynamodbav:"sk"`

  City     string `dynamodbav:"city"`
  Building string `dynamodbav:"building"`
  Floor    string `dynamodbav:"floor"`
  Room     string `dynamodbav:"room"`
}
```

As you can see `Sensor` knows nothing about underlying implementation. The `asItem` method is a transformation that makes sure that PK and SK are set in a proper way.

```go
func (s Sensor) asItem() sensorItem {
  return sensorItem{
    City:     s.City,
    ID:       "SENSOR#" + s.ID,
    SK:       "SENSORINFO",
    Building: s.Building,
    Floor:    s.Floor,
    Room:     s.Room,
  }
}
``` 

Notice also that I named Partition Key - PK, and Sort Key - SK. This is because we are using Single Table Design and  different items have their own meaning of the PK and SK. In this example SK has value `SENSORINFO`. It is a constant value. I am setting this that way so that we are able to distinguish a sensor and its readings. Now, back  to the implementation. The sensor is in the format that DynamoDB will understand. Next thing we need to take care of is uniqueness. We cannot register the same sensor twice and in order to achieve that we need a condition.

```go
expr, err := expression.NewBuilder().WithCondition(expression.AttributeNotExists(expression.Name("pk"))).Build()
```  

What it says is: "I am going to move further with the operation only if DynamoDB doesn't have an item with `pk` that I want to store in this operation".

```go
_, err = s.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
  TransactItems: []*dynamodb.TransactWriteItem{
    {
      Put: &dynamodb.Put{
        ConditionExpression:       expr.Condition(),
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),

        Item:      attrs,
        TableName: aws.String(s.table),
      },
    },
    {
      Put: &dynamodb.Put{
        Item: map[string]*dynamodb.AttributeValue{
            "pk": {S: aws.String("CITY#" + sensor.City)},
            "sk": {S: aws.String(fmt.Sprintf("LOCATION#%s#%s#%s", sensor.Building, sensor.Floor, sensor.Room))},
            "id": {S: aws.String(sensor.ID)},
        },
        TableName: aws.String(s.table),
      },
    },
  },
})
```

We want to put two items into the DynamoDB, sensor itself and the location. First Write Item has a condition that we defined and the other constructs the location. I decided to define it on the fly here because it's not important anywhere else.

Let's have a look at the error handling.

```go
if err != nil {
  _, ok := err.(*dynamodb.TransactionCanceledException)
  if ok {
    return errors.New("already registered")
  }
  return err
}
return nil
```

It needs to be handled explicitly because we need to verify whether transaction failed because of the failed condition or because something unexpected happened.

### [Sensor retrieval](#sensor-retrieval)

In order to retrieve sensor we need to use proper `SK` and `PK` which means we need to construct proper Composite Primary Key.

```go
map[string]*dynamodb.AttributeValue{
  "pk": {S: aws.String("SENSOR#" + id)},
  "sk": {S: aws.String("SENSORINFO")},
}
```
The ID needs to have the prefix, and SK needs to be the constant I choose to mark a sensor. If you want to see whole implementation of `Get` method please have a look [here](./v1/sensors.go). There is nothing interesting going on there - just simple data retrieval, so I am not repeating it here.

### [Saving a reading](#saving-a-reding)

Another fairly simple piece of code. It is just a PUT operation of a `Reading`. What is worth to talk about her is how data structure looks like.

```go
type Reading struct {
  SensorID string
  Value    string
  ReadAt   time.Time
}

type readingItem struct {
  SensorID string `dynamodbav:"pk"`
  Value    string `dynamodbav:"value"`
  ReadAt   string `dynamodbav:"sk"`
}
```

I used the same pattern as for `Sensor`. There is `Reading` that makes sense in the domain, and there is `readingItem` that defines how implementation is going to look like.

```go
func (r Reading) asItem() readingItem {
  return readingItem{
    SensorID: "SENSOR#" + r.SensorID,
    ReadAt:   "READ#" + r.ReadAt.Format(time.RFC3339),
    Value:    r.Value,
  }
}
```
This transformation makes sure that `PK` of an item begins with `SENSOR#` prefix. We need that because we want readings of the sensor and sensor itself to be in the same __Item Collection__. Item collection is collection of items that share the same Partition Key. We need that to be able to retrieve sensor and its latest readings with single query. Other thing that is going on here is formatting `SK` of an item in a way that will be sortable by time.

### [Retrieving the latest readings and the sensor](#retrieving-latest)

We will query two item types at the same time. We need some sort of condition.

```go
expr, err := expression.NewBuilder().WithKeyCondition(expression.KeyAnd(
  expression.KeyEqual(expression.Key("pk"), expression.Value("SENSOR#"+sensorID)),
  expression.KeyLessThanEqual(expression.Key("sk"), expression.Value("SENSORINFO")), 
)).Build()
```

Let's read it. Attribute `pk` is the ID prefixed with `SENSOR#`. This makes sense - we need to fetch whole item collection. Let's keep reading. Attribute `sk` needs to be less than or equal than `SENSORINFO`. Wait, what? We wanted to fetch the sensor and it's readings. How on earth such condition is going to achieve that? Bare with me.

| PK         | SK                    |
| ---        | ----                  |
| SENSOR#S1  | READ#2020-03-01-12:30 |
| SENSOR#S1  | READ#2020-03-01-12:31 |
| SENSOR#S1  | READ#2020-03-01-12:32 |
| SENSOR#S1  | READ#2020-03-01-12:33 |
| SENSOR#S1  | SENSORINFO            |

This is excerpt from the table that I showed you before but containing just Composite Primary Key. Items are sorted in ascending order by default. This means that readings are sorted from oldest to the newest, and after readings there is `SENSORINFO` because `S` comes after `R` in the alphabet. What we want to achieve is to read the data backwards starting from the item with `SENSORINFO` as `SK`. In order to read the data in this way we need to construct a query with parameter `ScanIndexForward` set to false.

```go
out, err := s.db.QueryWithContext(ctx, &dynamodb.QueryInput{
  ExpressionAttributeValues: expr.Values(),
  ExpressionAttributeNames:  expr.Names(),
  KeyConditionExpression:    expr.KeyCondition(),
  Limit:                     aws.Int64(last + 1),
  ScanIndexForward:          aws.Bool(false),
  TableName:                 aws.String(s.table),
})
```

Also, the limit is set to amount of last readings we want to retrieve increased by one so that we will retrieve information about the sensor as well.

What is going on at the end of the method is proper unmarshalling items into domain objects.
```go
var si sensorItem
err = dynamodbattribute.UnmarshalMap(out.Items[0], &si)

var ri []readingItem
err = dynamodbattribute.UnmarshalListOfMaps(out.Items[1:aws.Int64Value(out.Count)], &ri)

var readings []Reading
for _, r := range ri {
  readings = append(readings, r.asReading())
}
return si.asSensor(), readings, nil
```
We know for a fact that `Sensor` is first in the item collection, so it is unmarshalled as the `Sensor`. The rest of the items are treated as `Readings`.

### [Get sensors by location](#get-sensors-by-location)

As you remember in this version of implementation - the location is stored as an additional item. Method `GetSensors` accepts `Location` type that contains `City`, `Building`, `Floor` and `Room`. An item representing the location looks like this:

| PK          | SK                | ID  |
| ---         | ----              | --- |
| CITY#Poznań | LOCATION#A#2#13   |  S1 |

We need to build key condition that will point to `PK` which is just a `City` prefixed with `CITY#` and that has `SK` that begins with certain prefix. Depending on level of location precision - `SK` begins with shorter or longer prefix that specify from where we should get the sensors.

```go
expr, err := expression.NewBuilder().WithKeyCondition(expression.KeyAnd(
  expression.KeyEqual(expression.Key("pk"), expression.Value("CITY#"+location.City)),
  expression.KeyBeginsWith(expression.Key("sk"), location.asPath()), 
)).Build()
```

After building the condition expression we need to use it in the query:

```go
out, err := s.db.QueryWithContext(ctx, &dynamodb.QueryInput{
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),
  KeyConditionExpression:    expr.KeyCondition(),
  TableName:                 aws.String(s.table),
})
```

At the end I just prepare list of IDs that should be returned from the method.

```go
var ids []string
for _, i := range out.Items {
	ids = append(ids, aws.StringValue(i["id"].S))
}
return ids, nil
```

This is it. Complete code for first version of implementation is [here](./v1).

## [Implementation - version 2](#implementation-v2)

Second version of the implementation varies a little. The difference lays in how location is stored. In first version queryable location was just additional item. Second version uses Global Secondary Index for that purpose.

| PK         | SK                    | City   | Building  |  Floor |  Room  | Type | GSI_PK | GSI_SK |
| ---        | ----                  | ---    | ---       | ---    | ---    | ---  | ---    | ---    |
| SENSOR#S1  | SENSORINFO            | Poznań |  A        |   2    |  13    | Gas  | Poznań | A#2#13 |

Local Secondary Index cannot be used in this scenario because it would need to have the same Partition Key as Primary Key. Because we want to use different Partition Key - we need to use GSI.

I am going to show you only two methods because only they are different - registration of a sensor and retrieving sensors by the location.

### [Registration](#registration-v2)

`Sensor` type stays exactly the same because the domain sense of it doesn't change with implementation. However `sensorItem` is going to have two additional fields: `GSIPK` and `GSISK`.

```go
func (s Sensor) asItem() sensorItem {
  return sensorItem{
    City:     s.City,
    ID:       "SENSOR#" + s.ID,
    SK:       "SENSORINFO",
    Building: s.Building,
    Floor:    s.Floor,
    Room:     s.Room,
    GSIPK:    "CITY#" + s.City,
    GSISK:    fmt.Sprintf("LOCATION#%s#%s#%s", s.Building, s.Floor, s.Room),
  }
}
```
As you can see `GSIPK` and `GSISK` look exactly the same as the additional `location` item in the first version of implementation. It's the same information but inside`sensorItem`.

Registration itself holds exactly the same condition as before - which is to make sure that we are not introducing duplicated sensors. What changed is instead of using transactions - we use simple PUT operation.

```go
_, err = s.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
  ConditionExpression:       expr.Condition(),
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),

  Item:      attrs,
  TableName: aws.String(s.table),
})
``` 

Frankly speaking registration just got very boring. We transform the `Sensor` into `sensorItem` and drop it into the DynamoDB with a condition.

### [Get sensors by location](#get-sensors-by-location-v2)

This method changed just slightly compared to the first version. Let's have a look at the key condition.
```go
expr, err := expression.NewBuilder().WithKeyCondition(expression.KeyAnd(
  expression.KeyEqual(expression.Key("gsi_pk"), expression.Value("CITY#"+location.City)),
  expression.KeyBeginsWith(expression.Key("gsi_sk"), location.asPath()),
)).Build() 
```
It uses exactly the same mechanism as first version but instead of `pk` and `sk`, we use `gsi_pk` and `gsi_sk` when building key condition expression. What about the query? 
```go
out, err := s.db.QueryWithContext(ctx, &dynamodb.QueryInput{
  ExpressionAttributeNames:  expr.Names(),
  ExpressionAttributeValues: expr.Values(),
  KeyConditionExpression:    expr.KeyCondition(),
  TableName:                 aws.String(s.table),
  IndexName:                 aws.String("ByLocation"),
})
```
It didn't change much either. There is one additional bit which is `IndexName` that we used. This index has `GSI_PK` and `GSI_SK` as its key.

This is the whole difference between two versions.

## Summary

We covered a lot this time. Let me enumerate concepts that we used to make this work.

- Single Table Design
- Fetching two different item type with single query
- Modeling hierarchical data in DynamoDB
- Sparse Indexes
- Transactions

I hope you enjoyed this long journey. Also, I would like to invite you more than ever to fetch this repository and play with examples!