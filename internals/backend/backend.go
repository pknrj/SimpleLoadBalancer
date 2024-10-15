package backend

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL			*url.URL
	Alive 	 	bool
	mux 	 	sync.RWMutex
	RvsProxy 	*httputil.ReverseProxy
}

func (b *Backend) SetStatus(alive bool){
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) GetStatus() bool {
	  b.mux.Lock()
	  alive := b.Alive
	  b.mux.Unlock()
	  return alive
}




