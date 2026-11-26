package services

import (
	"sync"
	"time"
)

var pekerjaanLatar sync.WaitGroup

func JalankanLatar(fn func()) {
	pekerjaanLatar.Add(1)
	go func() {
		defer pekerjaanLatar.Done()
		fn()
	}()
}

func TungguLatar(batas time.Duration) bool {
	selesai := make(chan struct{})
	go func() {
		pekerjaanLatar.Wait()
		close(selesai)
	}()

	select {
	case <-selesai:
		return true
	case <-time.After(batas):
		return false
	}
}
