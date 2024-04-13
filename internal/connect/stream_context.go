package connect

type StreamContext struct {
	Sessions map[string]*Connection
	Preview  chan string
}
