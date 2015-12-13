package model

import (
	"log"
	"time"
	//"github.com/disintegration/imaging"
)

// Dispatcher function to spawn a number of workers
func StartDispatcher(nworkers int, WorkQueue chan File, log *log.Logger) {
	for i := 0; i<nworkers; i++ {
		go StartWorker(WorkQueue, log)
	}
}

func StartWorker(WorkQueue chan File, log *log.Logger) {
	for {
		select {
			case f := <-WorkQueue:
			        log.Print("Batch processing: " + f.Tag + ", " + f.Filename)
				// Simulate some processing time
				time.Sleep(10 * time.Second)
			        log.Print("Completed")
		}
	}
}
