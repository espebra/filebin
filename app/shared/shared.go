package shared

import (
	"net/http"
	"time"
	"log"
)

func PurgeURL(url string, log *log.Logger) error {
	timeout := time.Duration(2 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	// Invalidate the file
	req, err := http.NewRequest("PURGE", url, nil)
	if err != nil {
		log.Println("Unable to purge " + url + ": " + err.Error())
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Unable to purge " + url + ": " + err.Error())
		return err
	}

	log.Println("Purging " + url + ": " + resp.Status)

	return nil
}
