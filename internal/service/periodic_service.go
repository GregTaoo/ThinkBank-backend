package service

import (
	"log"
	"sync"
	"time"
)

type PeriodicTask struct {
	handler  func()
	interval time.Duration
}

type PeriodicService struct {
	mu    sync.Mutex
	tasks []PeriodicTask
}

var service = &PeriodicService{tasks: make([]PeriodicTask, 0)}

func RegisterPeriodicService(fn func(), interval time.Duration) {
	service.mu.Lock()
	service.tasks = append(service.tasks, PeriodicTask{fn, interval})
	service.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Fatal("Periodic service panic:", r)
					}
				}()
				fn()
			}()
		}
	}()
}

func RunAll() {
	service.mu.Lock()
	defer service.mu.Unlock()
	for _, task := range service.tasks {
		go task.handler()
	}
}
