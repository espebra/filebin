package log

import (
	"fmt"
	"time"
	//"net/http"
	//"log"
	//"io"
	"os"
	"github.com/espebra/filebin/app/config"
)

func writeLog(level string, text string) {
	t := time.Now()
	text = t.Format("Mon Jan _2 15:04:05 MST 2006") + " [" + level + "] " + text + "\n"

	f, err := os.OpenFile(config.Global.Logfile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		fmt.Print("Error opening file: %v", err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		fmt.Print("Error writing to file: %v", err)
	}

	if config.Global.Verbose {
		fmt.Print(text)
	}
}

func Error(text string) {
    writeLog("ERROR", text)
}

func Info(text string) {
    writeLog("INFO", text)
}

func Fatal(text string) {
    writeLog("FATAL", text)
    fmt.Println(text)
    os.Exit(2)
}
