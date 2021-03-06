package shootout

type Competitor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Health int    `json:"health"`
	Damage int    `json:"damage"`
}

func (c *Competitor) IsZero() bool {
	return Competitor{} == *c
}

type Round struct {
	Competitors map[string]*Competitor
}

type Shot struct {
	From string `json:"from"`
	To   string `json:"to"`
}
