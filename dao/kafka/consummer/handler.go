package consummer

// Handler 处理msg的程序
type Handler interface {
	WorkHandler(msg []byte)
}
