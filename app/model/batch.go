package model

import (
	"log"
	"math/rand"
	"time"
	//"github.com/disintegration/imaging"
)

// Dispatcher function to spawn a number of workers
func StartDispatcher(nworkers int, CacheInvalidation bool, WorkQueue chan File, log *log.Logger) {
	for i := 0; i < nworkers; i++ {
		go StartWorker(CacheInvalidation, WorkQueue, log)
	}
}

func StartWorker(CacheInvalidation bool, WorkQueue chan File, log *log.Logger) {
	var err error
	for {
		select {
		case f := <-WorkQueue:
			startTime := time.Now().UTC()

			jobId := "b-" + randomString(5) + " "
			log.SetPrefix(jobId)

			log.Print("Batch process starting: " + f.Bin + ", " + f.Filename)
			// Simulate some processing time
			if f.MediaType() == "image" {
				err = f.GenerateImage(115, 115, true)
				if err != nil {
					log.Print(err)
				}
				err = f.GenerateImage(1140, 0, false)
				if err != nil {
					log.Print(err)
				}

				if CacheInvalidation {
					if err := f.Purge(); err != nil {
						log.Print(err)
					}
				}
			}
			finishTime := time.Now().UTC()
			elapsedTime := finishTime.Sub(startTime)
			log.Println("Completed in: " + elapsedTime.String())
		}
	}
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
