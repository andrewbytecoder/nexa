package gops

import (
	"bytes"
	"fmt"
	"github.com/google/gops/goprocess"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

var develRe = regexp.MustCompile(`devel\s+\+\w+`)

func processes() {
	ps := goprocess.FindAll()

	var maxPID, maxPPID, maxExec, maxVersion int
	for i, p := range ps {
		ps[i].BuildVersion = shortenVersion(p.BuildVersion)
		maxPID = max(maxPID, len(strconv.Itoa(p.PID)))
		maxPPID = max(maxPPID, len(strconv.Itoa(p.PPID)))
		maxExec = max(maxExec, len(p.Exec))
		maxVersion = max(maxVersion, len(ps[i].BuildVersion))

	}

	for _, p := range ps {
		buf := bytes.NewBuffer(nil)
		pid := strconv.Itoa(p.PID)
		_, _ = fmt.Fprint(buf, pad(pid, maxPID))
		_, _ = fmt.Fprint(buf, " ")
		ppid := strconv.Itoa(p.PPID)
		_, _ = fmt.Fprint(buf, pad(ppid, maxPPID))
		_, _ = fmt.Fprint(buf, " ")
		_, _ = fmt.Fprint(buf, pad(p.Exec, maxExec))
		if p.Agent {
			_, _ = fmt.Fprint(buf, "*")
		} else {
			_, _ = fmt.Fprint(buf, " ")
		}
		_, _ = fmt.Fprint(buf, " ")
		_, _ = fmt.Fprint(buf, pad(p.BuildVersion, maxVersion))
		_, _ = fmt.Fprint(buf, " ")
		_, _ = fmt.Fprint(buf, p.Path)
		_, _ = fmt.Fprintln(buf)
		_, _ = buf.WriteTo(os.Stdout)
	}
}

func shortenVersion(v string) string {
	if !strings.HasPrefix(v, "devel") {
		return v
	}
	results := develRe.FindAllString(v, 1)
	if len(results) == 0 {
		return v
	}
	return results[0]
}

func pad(s string, total int) string {
	if len(s) >= total {
		return s
	}
	return s + strings.Repeat(" ", total-len(s))
}

// ProcessCommand displays information about a Go process.
func ProcessCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "process <pid> [period]",
		Aliases: []string{"pid", "proc"},
		Short:   "Prints information about a Go process.",
		// stop printing usage when the command errors
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProcessInfo(args)
		},
	}
}

// ProcessInfo takes arguments starting with pid|:addr and grabs all kinds of
// useful Go process information.
func ProcessInfo(args []string) error {
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("error parsing the first argument: %w", err)
	}

	var period time.Duration
	if len(args) >= 2 {
		period, err = time.ParseDuration(args[1])
		if err != nil {
			secs, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("error parsing the second argument: %w", err)
			}
			period = time.Duration(secs) * time.Second
		}
	}

	processInfo(pid, period)
	return nil
}

func processInfo(pid int, period time.Duration) {
	if period < 0 {
		log.Fatalf("Cannot determine CPU usage for negative duration %v", period)
	}
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		log.Fatalf("Cannot read process info: %v", err)
	}
	if v, err := p.Parent(); err == nil {
		fmt.Printf("parent PID:\t%v\n", v.Pid)
	}
	if v, err := p.NumThreads(); err == nil {
		fmt.Printf("threads:\t%v\n", v)
	}
	if v, err := p.MemoryPercent(); err == nil {
		fmt.Printf("memory usage:\t%.3f%%\n", v)
	}
	if v, err := p.CPUPercent(); err == nil {
		fmt.Printf("cpu usage:\t%.3f%%\n", v)
	}
	if period > 0 {
		if v, err := cpuPercentWithinTime(p, period); err == nil {
			fmt.Printf("cpu usage (%v):\t%.3f%%\n", period, v)
		}
	}
	if v, err := p.Username(); err == nil {
		fmt.Printf("username:\t%v\n", v)
	}
	if v, err := p.Cmdline(); err == nil {
		fmt.Printf("cmd+args:\t%v\n", v)
	}
	if v, err := elapsedTime(p); err == nil {
		fmt.Printf("elapsed time:\t%v\n", v)
	}
	if v, err := p.Connections(); err == nil {
		if len(v) > 0 {
			for _, conn := range v {
				fmt.Printf("local/remote:\t%v:%v <-> %v:%v (%v)\n",
					conn.Laddr.IP, conn.Laddr.Port, conn.Raddr.IP, conn.Raddr.Port, conn.Status)
			}
		}
	}
}

// cpuPercentWithinTime return how many percent of the CPU time this process uses within given time duration
func cpuPercentWithinTime(p *process.Process, t time.Duration) (float64, error) {
	cput, err := p.Times()
	if err != nil {
		return 0, err
	}
	time.Sleep(t)
	cput2, err := p.Times()
	if err != nil {
		return 0, err
	}
	return 100 * (cput2.Total() - cput.Total()) / t.Seconds(), nil
}

// elapsedTime shows the elapsed time of the process indicating how long the
// process has been running for.
func elapsedTime(p *process.Process) (string, error) {
	crtTime, err := p.CreateTime()
	if err != nil {
		return "", err
	}
	etime := time.Since(time.Unix(crtTime/1000, 0))
	return fmtEtimeDuration(etime), nil
}

// fmtEtimeDuration formats etime's duration based on ps' format:
// [[DD-]hh:]mm:ss
// format specification: http://linuxcommand.org/lc3_man_pages/ps1.html
func fmtEtimeDuration(d time.Duration) string {
	days := d / (24 * time.Hour)
	hours := d % (24 * time.Hour)
	minutes := hours % time.Hour
	seconds := math.Mod(minutes.Seconds(), 60)
	var b strings.Builder
	if days > 0 {
		_, err := fmt.Fprintf(&b, "%02d-", days)
		if err != nil {
			return ""
		}
	}
	if days > 0 || hours/time.Hour > 0 {
		_, err := fmt.Fprintf(&b, "%02d:", hours/time.Hour)
		if err != nil {
			return ""
		}
	}
	_, _ = fmt.Fprintf(&b, "%02d:", minutes/time.Minute)
	_, _ = fmt.Fprintf(&b, "%02.0f", seconds)
	return b.String()
}
