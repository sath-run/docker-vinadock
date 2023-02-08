package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

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

func appendConfig() error {
	configPath := "/data/config.txt"
	data, err := os.ReadFile(configPath)
	if !errors.Is(err, os.ErrNotExist) && err != nil {
		return err
	}

	configs := []string{}
	configMap := map[string]string{}
	regConfig, err := regexp.Compile(`(\w+)\s*=\s*(\S+)`)
	if err != nil {
		return err
	}

	for _, config := range strings.Split(string(data), "\n") {
		matches := regConfig.FindStringSubmatch(config)
		if len(matches) == 3 {
			configMap[matches[1]] = matches[2]
			configs = append(configs, config)
		}
	}

	if len(configMap["ligand"]) == 0 {
		configMap["ligand"] = "/data/ligand.pdbqt"
		configs = append(configs, fmt.Sprintf("ligand = %s", configMap["ligand"]))
	}
	if len(configMap["receptor"]) == 0 {
		configMap["receptor"] = "/data/receptor.pdbqt"
		configs = append(configs, fmt.Sprintf("receptor = %s", configMap["receptor"]))
	}
	if len(configMap["out"]) == 0 {
		configMap["out"] = "/data/output.pdbqt"
		configs = append(configs, fmt.Sprintf("out = %s", configMap["out"]))
	}
	if len(configMap["cpu"]) == 0 {
		cores := runtime.NumCPU()
		configs = append(configs, fmt.Sprintf("cpu = %d", cores))
	}
	if len(configMap["exhaustiveness"]) == 0 {
		configs = append(configs, "exhaustiveness = 32")
	}

	boxKeys := []string{"center_x", "center_y", "center_z", "size_x", "size_y", "size_z"}
	boxParamMissing := false
	for _, key := range boxKeys {
		if len(configMap[key]) == 0 {
			boxParamMissing = true
			break
		}
	}
	if boxParamMissing {
		gpfPath := "/data/receptor.gpf"
		cmd := exec.Command("prepare_gpf.py", "-l", configMap["ligand"], "-r",
			configMap["receptor"], "-o", gpfPath, "-y")

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return errors.WithMessage(err, stderr.String())
		}
		data, err := os.ReadFile(gpfPath)
		if err != nil {
			return err
		}
		regNpts, err := regexp.Compile(`npts (\d+) (\d+) (\d+)`)
		if err != nil {
			return err
		}
		regSpacing, err := regexp.Compile(`spacing (\S+)`)
		if err != nil {
			return err
		}
		regCenter, err := regexp.Compile(`gridcenter (\S+) (\S+) (\S+)`)
		if err != nil {
			return err
		}

		center_x, center_y, center_z := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		size_x, size_y, size_z := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		spacing := math.MaxFloat64

		for _, line := range strings.Split(string(data), "\n") {
			if matches := regNpts.FindStringSubmatch(line); len(matches) == 4 {
				if size_x, err = strconv.ParseFloat(matches[1], 64); err != nil {
					return err
				}
				if size_y, err = strconv.ParseFloat(matches[2], 64); err != nil {
					return err
				}
				if size_z, err = strconv.ParseFloat(matches[3], 64); err != nil {
					return err
				}
			}
			if matches := regSpacing.FindStringSubmatch(line); len(matches) == 2 {
				if spacing, err = strconv.ParseFloat(matches[1], 64); err != nil {
					return err
				}
			}
			if matches := regCenter.FindStringSubmatch(line); len(matches) > 0 {
				if center_x, err = strconv.ParseFloat(matches[1], 64); err != nil {
					return err
				}
				if center_y, err = strconv.ParseFloat(matches[2], 64); err != nil {
					return err
				}
				if center_z, err = strconv.ParseFloat(matches[3], 64); err != nil {
					return err
				}
			}
		}

		for _, val := range []float64{center_x, center_y, center_z, size_x, size_y, size_z, spacing} {
			if val == math.MaxFloat64 {
				return errors.Errorf("error parsing gpf: %f %f %f %f %f %f %f",
					center_x, center_y, center_z, size_x, size_y, size_z, spacing)
			}
		}

		size_x *= spacing
		size_y *= spacing
		size_z *= spacing

		if len(configMap["center_x"]) == 0 {
			configs = append(configs, fmt.Sprintf("center_x = %f", center_x))
		}
		if len(configMap["center_y"]) == 0 {
			configs = append(configs, fmt.Sprintf("center_y = %f", center_y))
		}
		if len(configMap["center_z"]) == 0 {
			configs = append(configs, fmt.Sprintf("center_z = %f", center_z))
		}
		if len(configMap["size_x"]) == 0 {
			configs = append(configs, fmt.Sprintf("size_x = %f", size_x))
		}
		if len(configMap["size_y"]) == 0 {
			configs = append(configs, fmt.Sprintf("size_y = %f", size_y))
		}
		if len(configMap["size_z"]) == 0 {
			configs = append(configs, fmt.Sprintf("size_z = %f", size_z))
		}
	}

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(strings.Join(configs, "\n") + "\n")
	if err != nil {
		return err
	}

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

	os.Remove("/data/output.log")
	outlog, err := os.OpenFile("/data/output.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("can't create sath log file: %+v\n", err)
	}
	defer outlog.Close()

	for {
		n, err := stdoutIn.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			for _, b := range d {
				if b == byte('*') {
					setProgress(&progress, progress+98.0/51.0, stdout)
				}
			}
			if _, err := outlog.Write(d); err != nil {
				return errors.WithStack(err)
			}
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

	os.Remove("/data/sath.log")
	stdout, err := os.OpenFile("/data/sath.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("can't create sath log file: %+v\n", err)
	}
	defer stdout.Close()

	os.Remove("/data/sath.err")
	stderr, err := os.OpenFile("/data/sath.err", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("can't create sath err file: %+v\n", err)
	}
	defer stderr.Close()

	err = appendConfig()
	if err != nil {
		errStr := fmt.Sprintf("%+v\n", err)
		fmt.Println(errStr)
		stderr.WriteString(errStr)
		os.Exit(1)
	}
	err = runVinaDock(stdout, program)
	if err != nil {
		errStr := fmt.Sprintf("%+v\n", err)
		fmt.Println(errStr)
		stderr.WriteString(errStr)
		os.Exit(1)
	}
}
