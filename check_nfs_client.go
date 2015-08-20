package main

import (
	"flag"
	"fmt"
	"github.com/kballard/go-shellquote"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	debug     bool
	buildtime string
)

type NagiosResult struct {
	exitCode  int
	text      string
	perfdata  string
	multiline []string
}

type ExecResult struct {
	returnCode int
	output     string
}

// Debugf is a helper function for debug logging if mainCfgSection["debug"] is set
func Debugf(s string) {
	if debug != false {
		fmt.Println("DEBUG " + fmt.Sprint(s))
	}
}

// nagiosExit uses the NagiosResult struct to output Nagios plugin compatible output and exit codes
func nagiosExit(nr NagiosResult) {
	text := nr.text
	exitCode := nr.exitCode
	switch {
	case nr.exitCode == 0:
		text = "OK: " + nr.text
		exitCode = nr.exitCode
	case nr.exitCode == 1:
		text = "WARNING: " + nr.text
		exitCode = nr.exitCode
	case nr.exitCode == 2:
		text = "CRITICAL: " + nr.text
		exitCode = nr.exitCode
	case nr.exitCode == 3:
		text = "UNKNOWN: " + nr.text
		exitCode = nr.exitCode
	default:
		text = "UNKNOWN: Exit code '" + string(nr.exitCode) + "'undefined :" + nr.text
		exitCode = 3
	}

	if len(nr.multiline) > 0 {
		multiline := ""
		for _, l := range nr.multiline {
			multiline = multiline + l + "\n"
		}
		fmt.Printf("%s|%s\n%s\n", text, nr.perfdata, multiline)
	} else {
		fmt.Printf("%s|%s\n", text, nr.perfdata)
	}
	os.Exit(exitCode)
}

func executeCommand(command string, timeout int, allowFail bool) ExecResult {
	Debugf("Executing " + command)
	parts := strings.SplitN(command, " ", 2)
	cmd := parts[0]
	cmdArgs := []string{}
	if len(parts) > 1 {
		args, err := shellquote.Split(parts[1])
		if err != nil {
			Debugf("executeCommand(): err: " + fmt.Sprint(err))
			os.Exit(1)
		} else {
			cmdArgs = args
		}
	}

	before := time.Now()
	out, err := exec.Command(cmd, cmdArgs...).CombinedOutput()
	duration := time.Since(before).Seconds()
	er := ExecResult{0, string(out)}
	if msg, ok := err.(*exec.ExitError); ok { // there is error code
		er.returnCode = msg.Sys().(syscall.WaitStatus).ExitStatus()
	}
	Debugf("Executing " + command + " took " + strconv.FormatFloat(duration, 'f', 5, 64) + "s")
	if err != nil && !allowFail {
		fmt.Println("executeCommand(): command failed: "+command, err)
		fmt.Println("executeCommand(): Output: " + string(out))
		os.Exit(1)
	}
	return er
}

func parseNfsstatOutput(s string) NagiosResult {
	perfdata := ""
	var multiline []string
	reMetric := regexp.MustCompile("\\s+([a-z]+):\\s+(\\d+)")
	data := strings.Split(s, "\n")
	if len(data) == 1 {
		return NagiosResult{text: "nfsstat output was empty", exitCode: 1, perfdata: ""}
	}
	for _, l := range data {
		if len(l) == 0 {
			continue
		}
		Debugf("line: " + l)
		if m := reMetric.FindStringSubmatch(l); len(m) > 1 {
			Debugf("metric: " + m[1] + "value: " + m[2])
			perfdata = perfdata + m[1] + "=" + m[2] + "c "
			multiline = append(multiline, m[0])
		}
	} // close for loop over output
	Debugf(perfdata)
	return NagiosResult{text: "nfsstat output successfully parsed", exitCode: 0, perfdata: perfdata, multiline: multiline}
}

func main() {
	var (
		debugFlag   = flag.Bool("debug", false, "log debug output, defaults to false")
		versionFlag = flag.Bool("version", false, "show build time and version number")
	)
	flag.Parse()

	debug = *debugFlag

	if *versionFlag {
		fmt.Println("check_nfs_client Version 0.1 Build time:", buildtime, "UTC")
		os.Exit(0)
	}

	nr := NagiosResult{exitCode: 3, text: "uncatched case", perfdata: ""}

	er := executeCommand("nfsstat -c -l", 1, false)
	nr = parseNfsstatOutput(er.output)

	nagiosExit(nr)

}
