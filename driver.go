package main

// driver program which runs uses the worker_pools to do task.
// The driver program has a task defined on a Type(T) such that T implements Work function
import (
	"flag"
	"log"
	worker "practice/main/worker"
	"runtime"
	"strings"
	"time"
)

type test string

// implement a thread intensive task on a string value
func (t *test) Work() {
	cnt := 0
	idx := 0
	for idx < 1000 {
		for idx := range *t {
			if strings.ContainsAny(string((*t)[idx]), "aeiou") {
				cnt++
			}
		}
		idx++
	}
}

func main() {
	numRoutines := flag.Int("workers", 1, "Number of workers")
	flag.Parse()
	log.Println("number of logical cpus-", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)

	// Initialize a new pool structure
	pool, err := worker.New(*numRoutines)
	if err != nil {
		log.Fatal(err)
	}
	stime := time.Now()

	// A sample work intensive task on a slice of strings
	{
		len := 100000
		t := make([]test, len, len)

		for idx := range t {
			t[idx] = "aaaaaaa"
		}
		for idx := range t {
			pool.AddTask(&t[idx])
		}
	}

	// wait for all worker pools to end
	pool.Shutdown()

	// log the time required for computation
	log.Println(time.Now().Sub(stime))

}
