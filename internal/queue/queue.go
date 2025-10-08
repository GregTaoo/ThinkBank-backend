package queue

import (
	"log"
	"sync"
)

type Payload struct {
	ID   uint
	Path string
}

// Message 定义消息结构
type Message struct {
	Topic string
	Data  any
}

// Queue 全局队列
type Queue struct {
	topics map[string]chan Message
	lock   sync.RWMutex
}

// GlobalQueue 全局变量
var GlobalQueue = NewQueue()

// NewQueue 创建队列
func NewQueue() *Queue {
	return &Queue{
		topics: make(map[string]chan Message),
	}
}

func (q *Queue) CheckTopic(topic string) chan Message {
	q.lock.RLock()
	ch, exists := q.topics[topic]
	q.lock.RUnlock()

	if exists {
		return ch
	}

	q.lock.Lock()
	defer q.lock.Unlock()

	ch, exists = q.topics[topic]
	if !exists {
		ch = make(chan Message, 1000)
		q.topics[topic] = ch
	}
	return ch
}

// Produce 生产消息
func (q *Queue) Produce(topic string, data any) {
	ch := q.CheckTopic(topic)

	// 发送消息（非阻塞）
	select {
	case ch <- Message{Topic: topic, Data: data}:
	default:
		log.Printf("queue %s full, message dropped\n", topic)
	}
}

// RegisterConsumer 注册消费者，支持 n 个并发消费者
func (q *Queue) RegisterConsumer(topic string, handler func(Message), n int) {
	ch := q.CheckTopic(topic)

	for i := 0; i < n; i++ {
		go func() {
			for msg := range ch {
				// 安全调用 handler(msg) 的闭包，用于防止消费者 goroutine 因 panic 整个程序崩掉
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Fatal("Consumer panic:", r)
						}
					}()
					handler(msg)
				}()
			}
		}()
	}
}
