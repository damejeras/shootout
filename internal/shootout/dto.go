package shootout

type Competitor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Health int    `json:"health"`
	Damage int    `json:"damage"`
}

type Round struct {
	ID          int `json:"id"`
	Competitors map[string]*Competitor
}

type Shot struct {
	From string `json:"from"`
	To   string `json:"to"`
}
