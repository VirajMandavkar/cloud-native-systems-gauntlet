package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type TaskResult struct {
	Target string
	Error  error
	Stdout string
	Stderr string
}

var (
	commandFlag string
	targetsFlag []string
	timeoutFlag time.Duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "exec",
	Short: "A distributed task executor",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutFlag)
		defer cancel()

		results := make(chan TaskResult, len(targetsFlag))
		var wg sync.WaitGroup

		for _, target := range targetsFlag {
			wg.Add(1)
			go func(tgt string) {
				defer wg.Done()
				cmd := exec.CommandContext(ctx, "sh", "-c", commandFlag)
				cmd.Env = append(os.Environ(), "TARGET="+tgt)

				var stdoutBuf, stderrBuf bytes.Buffer
				cmd.Stdout = &stdoutBuf
				cmd.Stderr = &stderrBuf

				err := cmd.Run()

				var etr TaskResult
				etr.Target = tgt
				etr.Error = err
				etr.Stdout = stdoutBuf.String()
				etr.Stderr = stderrBuf.String()

				results <- etr
			}(target)
		}
		go func() {
			wg.Wait()
			close(results)
		}()

		for res := range results {

			fmt.Printf("Target Name: %s\n", res.Target)
			if res.Error != nil {
				fmt.Printf("Error: %s\n", res.Error)
			}

			fmt.Printf("Stdout: %s\n", res.Stdout)
			fmt.Printf("Stderr: %s\n", res.Stderr)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&commandFlag, "command", "c", "", "The shell command to run on all targets")
	rootCmd.Flags().StringSliceVarP(&targetsFlag, "targets", "t", []string{}, "Comma-separated list of dummy targets")
	rootCmd.Flags().DurationVarP(&timeoutFlag, "timeout", "o", 5*time.Second, "Global execution timeout duration")

	_ = rootCmd.MarkFlagRequired("command")
	_ = rootCmd.MarkFlagRequired("targets")
}
