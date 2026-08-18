package main

import (
	"fmt"
	"os"
	//"os/exec"
	//	"syscall"
)

func main() {
	if len(os.Args) < 3 {
		panic("usage: ./runtime run <command>")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("invalid command")
	}
}

func run() {
	fmt.Printf("Host: Setting up namespaces for %v\n", os.Args[2:])

	// YOUR TURN:
	// 1. Create an exec.Command that executes "/proc/self/exe" (a Linux trick that points to the currently running binary).
	// 2. Pass "child" as the first argument, followed by os.Args[2:] (the actual command the user wants to run).
	// 3. Set the command's SysProcAttr to use Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS
	// 4. Bind the command's Stdin, Stdout, and Stderr to the os equivalents so we can interact with it.
	// 5. Run the command and handle the error.
}

func child() {
	fmt.Printf("Container: Executing %v\n", os.Args[2:])

	// We will add the `chroot` and memory limits here later.

	// YOUR TURN:
	// 1. Create an exec.Command using os.Args[2] (the binary, e.g., "sh") and os.Args[3:] (the args).
	// 2. Bind the command's Stdin, Stdout, and Stderr to the os equivalents.
	// 3. Run the command and handle the error.
}
