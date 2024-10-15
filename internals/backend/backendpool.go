package backend

import (
	"log"
	"net"
	"net/url"
	"sync/atomic"
	"time"
)

type BackendPool struct {
	backends 	[]*Backend
	currIdx 	uint64
}

func (bp *BackendPool) AppendBackend(b *Backend){
	bp.backends = append(bp.backends, b)
}

// NextIndex atomically increase the counter and return an index
func (bp *BackendPool) NextIndex() int {
	return int(atomic.AddUint64(&bp.currIdx, uint64(1)) % uint64(len(bp.backends)))
}

func (bp *BackendPool) SetBackendStatus(burl *url.URL , alive bool) {
	for _ , b := range bp.backends {
		if b.URL.String() == burl.String() {
			b.SetStatus(alive)
			break
		}
	}
}


func (bp *BackendPool) GetNextServer() *Backend{
	nextIdx := bp.NextIndex()
	l := len(bp.backends) + nextIdx 
	for i := nextIdx; i < l; i++ {
		idx := i % len(bp.backends) 
		if bp.backends[idx].GetStatus() {
			if i != nextIdx {
				atomic.StoreUint64(&bp.currIdx, uint64(idx))
			}
			return bp.backends[idx]
		}
	}
	return nil
}

func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}

func (bp *BackendPool) HealthCheck() {
	for _, b := range bp.backends {
		status := "running"
		alive := isBackendAlive(b.URL)
		b.SetStatus(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}