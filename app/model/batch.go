package model

import (
	"log"
	"time"
	"math/rand"
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
		                startTime := time.Now().UTC()

				jobId := "b-" + randomString(5) + " "
				log.SetPrefix(jobId)

			        log.Print("Batch process starting: " + f.Tag + ", " + f.Filename)
				// Simulate some processing time
				if f.MediaType() == "image" {
					err := f.GenerateThumbnail()
					if err != nil {
						log.Print(err)
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
