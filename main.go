package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

type ProgressData struct {
	Progress float64 `json:"progress"`
}

type ProgressMessage struct {
	Format  string       `json:"format"`
	Version string       `json:"version"`
	Type    string       `json:"type"`
	Data    ProgressData `json:"data"`
}

func setProgress(progress *float64, newValue float64, stdout io.Writer) error {
	*progress = newValue
	message := ProgressMessage{
		Format:  "sath",
		Version: "1.0",
		Type:    "progress",
		Data: ProgressData{
			Progress: *progress,
		},
	}
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}
	stdout.Write(jsonData)
	stdout.Write([]byte("\n"))
	return nil
}

func runVinaDock(stdout io.Writer, program string) error {
	cmd := exec.Command(fmt.Sprintf("/vinadock/bin/%s", program), "--config", "/data/config.txt")

	config, err := os.ReadFile("/data/config.txt")
	if err != nil {
		return errors.WithStack(err)
	}
	fmt.Printf("program: %s\n", program)
	fmt.Println(string(config))

	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		return errors.WithStack(err)
	}

	stdoutErr, err := cmd.StderrPipe()
	if err != nil {
		return errors.WithStack(err)
	}

	err = cmd.Start()
	if err != nil {
		return errors.WithStack(err)
	}

	buf := make([]byte, 1024)
	var progress float64
	setProgress(&progress, 1.0, stdout)
	for {
		n, err := stdoutIn.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			for _, b := range d {
				if b == byte('*') {
					setProgress(&progress, progress+98.0/51.0, stdout)
				}
			}
			// if _, err := stdout.Write(d); err != nil {
			// 	return errors.WithStack(err)
			// }
			fmt.Print(string(d))
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			} else {
				return errors.WithStack(err)
			}
			break
		}
	}
	errMsg, _ := io.ReadAll(stdoutErr)

	if err = cmd.Wait(); err != nil {
		return errors.New(string(errMsg))
	}

	setProgress(&progress, 100.0, stdout)
	return nil
}

func main() {

	var program string
	flag.StringVar(&program, "program", "", "docking program")
	flag.Parse()

	stdout, err := os.OpenFile("/data/sath.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("can't create sath log file: %+v\n", err)
	}
	defer stdout.Close()

	stderr, err := os.OpenFile("/data/sath.err", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("can't create sath err file: %+v\n", err)
	}
	defer stderr.Close()

	err = runVinaDock(stdout, program)
	if err != nil {
		stderr.WriteString(fmt.Sprintf("%+v\n", err))
		os.Exit(1)
	}
}
