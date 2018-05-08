package godscache

import "log"

type Godscache struct {
}

func NewGodscache() *Godscache {
	g := &Godscache{}
	log.Printf("Instantiated new Godscache: %+v", g)
	return g
}
