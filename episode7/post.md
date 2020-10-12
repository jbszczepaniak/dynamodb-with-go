# DynamoDB with Go #7

Imagine that you are working for real estate company that offers office space for rent. All the buildings are those smart new ones with a lot of sensors measuring myriad of parameters. You need to design a system that will manage somehow these sensors.

## [Access patterns](#access-patterns)

First things first, before we open IDE we need to know what are we going to do with this application. Let's enumerate all actions that should be possible.

We need to be able to register the sensor. I want to introduce a new sensor into the system in a following way:
```go
Register(Sensor{ID: "humidity-sensor-1", City: "Poznan", Building: "A", Floor: "3", Room: "112"})
```
After succesful registration I expect that there will be possibility to write a new sensor reading.
```go
NewReading(Reading{SensorID: "humidity-sensor-1", Value: "0.67"})
```
Sensor is registered and there are some data points already in the DB. How can we display this to the user of the system? I would like to be able to get sensors from locations. For example, I would like to say:
- give me all sensors in Lisbon,
- give me all sensors in Berlin in building D on 4th floor,
- give me all sensors in Poznań in building A, 2nd floor in room 432.

I am thinking about the API that would look like this:
```go
GetSensors(Location{City: "Poznań", Building: "A", Floor: "2"})
GetSensors(Location{City: "Poznań", Building: "D", Floor: "1", Room: "123"})
```
Let's say I searched for: _"Poznań, Building A"_ and received list of 25 different sensors. In terms of UI - I want to click on given sensor and receive detailed information about the sensor with 10 latest readings of this sensor. I imagine a call:
```go
Get("carbon-monoxide-sensor-2", 10)
```

It seems simple enough. We have four different functions operating on two different types.

## [Table design](#table-design)

We know WHAT we want achieve. It's time to define the HOW. How can we implement that? I would like to make this episode an exercise for two different things:
1. Single Table Design,
2. modelling hierarchical data.

### [Single Table Design](#single-table-design)

The idea of single table design is to be able to fetch different types of entities with single query to the DynamoDB. Since there is no possibility to query many tables at the same time, different types needs to be stored together within single table.

### [Modeling hierarchical data](#modeling-hierarchical-data)

Being able to fetch data on different level of hierarchy requires some thought upfront. We are going to leverage ability to use `begins_with` method on the sort key.

### [Sensors Table](#sensors-table)

| PK         | SK                    | Value | City   | Building  |  Floor |  Room  | Type |
| ---        | ----                  | ---   | ---    | ---       | ---    | ---    | ---  |
| SENSOR#S1  | READ#2020-03-01-12:30 |   2   |        |           |        |        |      |
| SENSOR#S1  | READ#2020-03-01-12:31 |   3   |        |           |        |        |      |
| SENSOR#S1  | READ#2020-03-01-12:32 |   5   |        |           |        |        |      |
| SENSOR#S1  | READ#2020-03-01-12:33 |   3   |        |           |        |        |      |
| SENSOR#S1  | SENSORINFO            |       | Poznań |  A        |   2    |  13    | Gas  |
 
I sketched out first attempt to put data into the table. First thing to notice is that Partition Key and Sort Key don't have a name that conveys domain knowledge. They're just abbreviation and this is because in Single Table Design there are different types of entities, hence PK and SK can mean different things for a different item type.

This layout allows us to query a sensor and obtain detailed information together with the latest reads from it. This is why we are doing the Single Table Design. We only have to query the DynamoDB once to obtain different types of entities.

This isn't bad so far, but there is still one feature missing - querying for many sensors that share the same location. As I said before we are going to use `begins_with` on the Sort Key. I want to do queries like these (pseudocode):

```pseudocode
Query(PK="Poznań", SKbeginswith="A#-1") -> all sensors from garage in building A in Poznań
Query(PK="Berlin") -> all sensors in Berlin
Query(PK="Lisbon", SKbeginswith="F#3#102") -> all sensors in Lisbon in the building F, room 102 on 3rd floor.
```

In order to do that we need to introduce additional artificial attribute which is a concatenation of different attributes.

First thing that comes to mind is to simply write something like this to the table:

| PK              | SK               | ID  |
| ---             | ----             | --- |
| CITY#Poznań     | LOCATION#A#2#13  | S1  |

Now I can satisfy the requirement for querying sensors in the given location. One thing to remember though is that we have two items that need to be synchronized. Detailed information about the sensor, and its location. Registering a sensor and changing the location is more complicated with this approach because we need to change two items in the DynamoDB transactionally so that these pieces of information won't diverge. 

The other idea is to use Global Secondary Index (GSI).

| PK         | SK                    | Value | City   | Building  |  Floor |  Room  | Type | GSI_PK | GSI_SK |
| ---        | ----                  | ---   | ---    | ---       | ---    | ---    | ---  | ---    | ---    |
| SENSOR#S1  | READ#2020-03-01-12:30 |   2   |        |           |        |        |      |        |        |
| SENSOR#S1  | READ#2020-03-01-12:31 |   3   |        |           |        |        |      |        |        |
| SENSOR#S1  | READ#2020-03-01-12:32 |   5   |        |           |        |        |      |        |        |
| SENSOR#S1  | READ#2020-03-01-12:33 |   3   |        |           |        |        |      |        |        |
| SENSOR#S1  | SENSORINFO            |       | Poznań |  A        |   2    |  13    | Gas  | Poznań | A#2#13 |

Whenever we want to query for sensors in given location we use GSI to do that. Hidden here is yet another concept which is called the Sparse Index. This index is sparse because it contains only some of the items. When querying or scanning that index we won't get any item that has `READ#` prefix in the SK, because these items aren't in this index (because these items don't have value for GSI_PK and GSI_SK attributes).

Which approach is better? Additional item in a table is very simple approach that just works. One downside it has is that when sensor is being registered or it's location changes - we need to change two items transactionally and programmatic error can cause this data to diverge. On the other hand there is GSI. Its advantage is that we need to change only  single item at a time when location changes or when we register new sensor. You need to be aware however that indexes cost extra money and that data is copied to the GSI in a asynchronous way which further means that when reading data from GSI, strongly consistent reads are not an option. All the reads from GSI are eventually consistent.

# [Summary](#summary)

I think we've done enough work for now. Let me split this topic into 2 episodes. This one was about __DynamoDB__, next one will be more __with Go__. Nevertheless, to summarize, we know what our access patterns are, and we know how to implement that in two different ways. More over we learned what are cons and pros of each approach. What I propose for the next episode  is that first we will write unit tests that define behavior we want to obtain. Then we are going to implement both ways: with additional item and with GSI.