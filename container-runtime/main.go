package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
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

	//Synchronization pipe: the child MUST NOT processed (mount/chroot/exec) until we've confirmed its been placed in the cgroup. 
	//cmd.Start() returns before the child does any work, and cgroup membership is NOT retroactively applied to processes forked before the parent joins the cgroup
	// Without this barrier , the users shell process gets forked and inherits the default cgroup before we ever write to cgroup.rpocs - limits silently do nothing

	r, w, err := os.Pipe()
	if err != nil {
		panic("pipe failed: " + err.Error())
	}

	args := append([]string{"child"}, os.Args[2:]...)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{r} // read end becomes fd 3 inside child

	// PID-namespaced cgroup path — a static path would collide if two containers run concurrently, silently merging their limits.
	cgPath := fmt.Sprintf("/sys/fs/cgroup/phantom-%d", os.Getpid())

	if err := os.MkdirAll(cgPath, 0755); err != nil {
		panic("failed to create cgroup: " + err.Error())
	}
	defer os.Remove(cgPath)

	// Limits are set up BEFORE cmd.Start(), not after. Setting them post-start leaves a real window where the child runs completely unconstrained until the writes land.
	if err := os.WriteFile(cgPath+"/memory.max", []byte("500000000"), 0700); err != nil {
		panic("failed to write memory limit: " + err.Error())
	}


	if err := os.WriteFile(cgPath+"/memory.swap.max", []byte("0"), 0700); err != nil {
		fmt.Printf("Host Warning: could not disable swap: %v\n", err)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Host Error: failed to start child process: %v\n", err)
		os.Exit(1)
	}

	pid := fmt.Sprintf("%d", cmd.Process.Pid)
	if err := os.WriteFile(cgPath+"/cgroup.procs", []byte(pid), 0700); err != nil {
		panic("failed to assign process to cgroup: " + err.Error())
	}

	// Only now is it safe to let the child continue — it's confirmed
	// inside the cgroup, so anything it forks from here on inherits
	// the enforced limits correctly.
	w.Write([]byte{1})
	w.Close()

	if err := cmd.Wait(); err != nil {
		fmt.Printf("Host Error: process finished with error: %v\n", err)
	}

}

func child() {
	fmt.Printf("Container: Executing %v\n", os.Args[2:])

	// Block until the parent confirms cgroup placement is complete.
	// See sync pipe comment in run() — this is the other half of the
	// race condition fix.
	pipe := os.NewFile(3, "pipe")
	buf := make([]byte, 1)
	pipe.Read(buf)
	pipe.Close()

	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		panic("mount failed: " + err.Error())
	}

	if err := syscall.Chroot("./alpine-fs"); err != nil {
		panic("chroot failed: " + err.Error())
	}
	if err := syscall.Chdir("/"); err != nil {
		panic("chdir failed: " + err.Error())
	}

	if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
		panic("proc mount failed: " + err.Error())
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Container Error: failed to execute user command: %v\n", err)
		syscall.Unmount("proc", 0)
		os.Exit(1)
	}
}
