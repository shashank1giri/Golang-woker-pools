package worker

import (
	"container/list"
	"errors"
	"sync"
)

const (
	addRoutine = iota
	removeRoutine
)

// Any type using this worker pool should implement the Work() method of this interface
type Worker interface {
	Work()
}

type Pool struct {
	numWorkerRoutines int            // number of worker routines
	scheduler         chan Worker    // channel which takes the jobs from taskQueue and schedules it on the worker routines
	ctrl              chan int       // channel for increasing or decreasing the number of worker routines in system
	wg                sync.WaitGroup // for waiting until all the tasks have been completed
	mux               sync.Mutex     // for synchronizing the enqueue operation in taskQueue
	kill              chan struct{}  // channel which sends a kill signal to kill the routine picking it
	shutdown          chan struct{}  // shutting down the system - gracefully waits for the queued tasks to complete and kills the
	//existing go routines - it is done by closing this channel
	taskQueue *list.List // task Queue
}

func New(minRoutines int) (*Pool, error) {
	if minRoutines <= 0 {
		return nil, errors.New("numWorkerRoutines cannot be negative")
	}
	pool := Pool{
		numWorkerRoutines: minRoutines,
		scheduler:         make(chan Worker),
		ctrl:              make(chan int),
		kill:              make(chan struct{}),
		shutdown:          make(chan struct{}),
		taskQueue:         list.New(),
	}
	pool.manager()
	pool.add(minRoutines)
	return &pool, nil
}

func (p *Pool) AddTask(work Worker) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.taskQueue.PushBack(work)
}

func (p *Pool) Shutdown() {
	close(p.shutdown)
	p.wg.Wait()

}

// manager  for Worker Pool. manages spawning of new routines, killing already existing routines.
func (p *Pool) manager() {
	p.wg.Add(1)
	go func() {
		for {
			select {
			case <-p.shutdown: // if shutdown channel is fired

				// wait until the task queue is not empty
				if p.taskQueue.Len() != 0 {
					continue
				}

				// kill all the existing worker routines
				for i := 0; i < p.numWorkerRoutines; i++ {
					p.kill <- struct{}{}
				}
				p.wg.Done()
				return

			case c := <-p.ctrl: // if ctrl channel is fired. i.e either for spawning or removing a routine
				switch c {
				case addRoutine:
					p.wg.Add(1)
					go p.work()
				case removeRoutine:
					p.kill <- struct{}{}

				}
			}
		}
	}()

	go func() {
		p.wg.Add(1)
		for {
			select {
			case <-p.shutdown:
				p.wg.Done()
				return
			default:
				for p.taskQueue.Len() != 0 {
					elem := p.taskQueue.Front()
					p.scheduler <- elem.Value.(Worker)
					p.taskQueue.Remove(elem)
				}
			}
		}
	}()

}

// work for each routine
func (p *Pool) work() {
loop:
	for {
		select {
		case task := <-p.scheduler:
			task.Work()
		case <-p.kill:
			p.wg.Done()
			break loop
		}
	}
}

// add routines to the pool
func (p *Pool) add(numRoutines int) {
	var val = addRoutine
	if numRoutines < 0 {
		val = removeRoutine
		numRoutines *= -1
	}

	// the manager reads ctrl channel and takes necessary actions for maintaining the num of routines
	for i := 0; i < numRoutines; i++ {
		p.ctrl <- val
	}
}
