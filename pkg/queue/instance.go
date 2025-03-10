// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package queue

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"

	"istio.io/istio/pkg/backoff"
	"istio.io/pkg/log"
)

// Task to be performed.
type Task func() error

// Instance of work tickets processed using a rate-limiting loop
type Instance interface {
	// Push a task.
	Push(task Task)
	// Run the loop until a signal on the channel
	Run(<-chan struct{})

	// Closed returns a chan that will be signaled when the Instance has stopped processing tasks.
	Closed() <-chan struct{}
}

type queueImpl struct {
	delay        time.Duration
	retryBackoff *backoff.ExponentialBackOff
	tasks        []Task
	cond         *sync.Cond
	closing      bool
	closed       chan struct{}
	closeOnce    *sync.Once
	id           string
}

// NewQueue instantiates a queue with a processing function
func NewQueue(errorDelay time.Duration) Instance {
	return NewQueueWithID(errorDelay, rand.String(10))
}

func NewQueueWithID(errorDelay time.Duration, name string) Instance {
	return &queueImpl{
		delay:     errorDelay,
		tasks:     make([]Task, 0),
		closing:   false,
		closed:    make(chan struct{}),
		closeOnce: &sync.Once{},
		cond:      sync.NewCond(&sync.Mutex{}),
		id:        name,
	}
}

func (q *queueImpl) Push(item Task) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	if !q.closing {
		q.tasks = append(q.tasks, item)
	}
	q.cond.Signal()
}

func (q *queueImpl) Closed() <-chan struct{} {
	return q.closed
}

// get blocks until it can return a task to be processed. If shutdown = true,
// the processing go routine should stop.
func (q *queueImpl) get() (task Task, shutdown bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// wait for closing to be set, or a task to be pushed
	for !q.closing && len(q.tasks) == 0 {
		q.cond.Wait()
	}

	if q.closing {
		// We must be shutting down.
		return nil, true
	}
	task = q.tasks[0]
	// Slicing will not free the underlying elements of the array, so explicitly clear them out here
	q.tasks[0] = nil
	q.tasks = q.tasks[1:]
	return task, false
}

func (q *queueImpl) processNextItem() bool {
	// Wait until there is a new item in the queue
	task, shuttingdown := q.get()
	if shuttingdown {
		return false
	}

	// Run the task.
	if err := task(); err != nil {
		delay := q.delay
		log.Infof("Work item handle failed (%v), retry after delay %v", err, delay)
		time.AfterFunc(delay, func() {
			q.Push(task)
		})
	}
	return true
}

func (q *queueImpl) Run(stop <-chan struct{}) {
	log.Debugf("started queue %s", q.id)
	defer func() {
		q.closeOnce.Do(func() {
			log.Debugf("closed queue %s", q.id)
			close(q.closed)
		})
	}()
	go func() {
		<-stop
		q.cond.L.Lock()
		q.cond.Signal()
		q.closing = true
		q.cond.L.Unlock()
	}()

<<<<<<< HEAD
	for {
		q.cond.L.Lock()
		for !q.closing && len(q.tasks) == 0 {
			q.cond.Wait()
		}

		if len(q.tasks) == 0 {
			q.cond.L.Unlock()
			// We must be shutting down.
			return
		}

		var task Task
		task, q.tasks = q.tasks[0], q.tasks[1:]
		q.cond.L.Unlock()

		if err := task(); err != nil {
			log.Infof("Work item handle failed (%v), retry after delay %v", err, q.delay)
			// 如果执行失败在放到队列中继续消费
			time.AfterFunc(q.delay, func() {
				q.Push(task)
			})
		}
=======
	for q.processNextItem() {
>>>>>>> 05ba771af6cd839e06483c3157ad910cb664da07
	}
}
