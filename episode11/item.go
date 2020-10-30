package episode12

type Key struct {
	PK string `json:"pk"`
	SK string `json:"sk"`
}

type Item struct {
	Key
	A string `json:"a"`
	B string `json:"b"`
}

