package episode12

type Key struct {
	PK string `dynamodbav:"pk"`
	SK string `dynamodbav:"sk"`
}

type Item struct {
	Key
	A string `dynamodbav:"a"`
	B string `dynamodbav:"b"`
}
