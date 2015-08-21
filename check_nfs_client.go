package main

import (
	"flag"
	"fmt"
	"github.com/xorpaul/go-nagios"
	"os"
	"regexp"
	"strings"
)

var (
	debug     bool
	buildtime string
)

func parseNfsstatOutput(s string) nagios.NagiosResult {
	perfdata := ""
	var multiline []string
	reMetric := regexp.MustCompile("\\s+([a-z]+):\\s+(\\d+)")
	data := strings.Split(s, "\n")
	if len(data) == 1 {
		return nagios.NagiosResult{Text: "nfsstat output was empty", ExitCode: 1, Perfdata: ""}
	}
	for _, l := range data {
		if len(l) == 0 {
			continue
		}
		nagios.Debugf("line: " + l)
		if m := reMetric.FindStringSubmatch(l); len(m) > 1 {
			nagios.Debugf("metric: " + m[1] + "value: " + m[2])
			perfdata = perfdata + m[1] + "=" + m[2] + "c "
			multiline = append(multiline, m[0])
		}
	} // close for loop over output
	nagios.Debugf(perfdata)
	return nagios.NagiosResult{Text: "nfsstat output successfully parsed", ExitCode: 0, Perfdata: perfdata, Multiline: multiline}
}

func main() {
	var (
		debugFlag   = flag.Bool("debug", false, "log debug output, defaults to false")
		versionFlag = flag.Bool("version", false, "show build time and version number")
	)
	flag.Parse()

	debug = *debugFlag
	nagios.Debug = *debugFlag

	if *versionFlag {
		fmt.Println("check_nfs_client Version 0.1 Build time:", buildtime, "UTC")
		os.Exit(0)
	}

	nr := nagios.NagiosResult{ExitCode: 3, Text: "uncatched case", Perfdata: ""}

	er := nagios.ExecuteCommand("nfsstat -c -l", 1, false)
	nr = parseNfsstatOutput(er.Output)

	nagios.NagiosExit(nr)

}
