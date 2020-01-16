package model

import (
	"github.com/espebra/filebin/app/backend/fs"
	"github.com/espebra/filebin/app/shared"
	"math/rand"
	"time"
)

// Dispatcher function to spawn a number of workers
func StartDispatcher(nworkers int, WorkQueue chan Job, backend *fs.Backend) {
	for i := 0; i < nworkers; i++ {
		go StartWorker(WorkQueue, backend)
	}
}

func StartWorker(WorkQueue chan Job, backend *fs.Backend) {
	for {
		select {
		case j := <-WorkQueue:
			startTime := time.Now().UTC()

			if err := backend.GenerateThumbnail(j.Bin, j.Filename, 115, 115, true); err != nil {
				j.Log.Println(err.Error())
				break
			}

			if err := backend.GenerateThumbnail(j.Bin, j.Filename, 1140, 0, false); err != nil {
				j.Log.Println(err.Error())
				break
			}

			if j.Cfg.CacheInvalidation {
				links := backend.GenerateLinks(j.Bin, j.Filename)
				for _, l := range links {
					if err := shared.PurgeURL(l.Href, j.Log); err != nil {
						j.Log.Println(err)
					}
				}
			}

			finishTime := time.Now().UTC()
			elapsedTime := finishTime.Sub(startTime)
			j.Log.Println("Batch job completed: " + j.Bin + "/" + j.Filename + " (" + elapsedTime.String() + ")")
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
