package transfer

import (
	"io"
	"sync"
)

const copyBufferSize = 4 << 20 // 4 MiB — important for jump-host download throughput

var copyBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, copyBufferSize)
		return &buf
	},
}

func fastCopy(dst io.Writer, src io.Reader) (int64, error) {
	buf := copyBufferPool.Get().(*[]byte)
	defer copyBufferPool.Put(buf)
	return io.CopyBuffer(dst, src, *buf)
}

func progressTotal(current, estimated int64) int64 {
	if estimated <= 0 {
		return 0
	}
	if current > estimated {
		return current + current/5
	}
	return estimated
}
