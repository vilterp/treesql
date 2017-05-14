package treesql

// https://github.com/jnwhiteh/golang-examples/blob/master/src/jnwhiteh.net/buffers/bytebuffer.go

type ByteBuffer struct {
	arr []byte
}

func NewByteBuffer() *ByteBuffer {
	var b *ByteBuffer = new(ByteBuffer)
	b.arr = make([]byte, 0, 2)
	return b
}

func (b *ByteBuffer) Write(p []byte) (n int, err error) {
	b.arr = append(b.arr, p...)
	return len(p), nil
}

func (b *ByteBuffer) WriteString(s string) (n int, err error) {
	conv := []byte(s)
	b.arr = append(b.arr, conv...)
	return len(conv), nil
}

func (b *ByteBuffer) GetBytes() (p []byte) {
	return b.arr
}

func (b *ByteBuffer) GetString() (s string) {
	return string(b.arr)
}
