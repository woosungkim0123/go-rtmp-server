package internal

type StreamContext struct {
	Sessions map[string]*Connection
	Preview  chan string
}

func (ctx *StreamContext) set(key string, c *Connection) {
	ctx.Sessions[key] = c
}

func (ctx *StreamContext) get(key string) *Connection {
	return ctx.Sessions[key]
}
