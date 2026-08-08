package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Printf("pid=%d\n", os.Getpid())
	fmt.Printf("ppid=%d\n", os.Getppid())
	fmt.Printf("args=%v\n", os.Args)

	if value, ok := os.LookupEnv("APP_MODE"); ok {
		fmt.Println("APP_MODE=", value)
	} else {
		fmt.Println("APP_MODE is not set")
	}

	fmt.Println("type one line and press Enter:")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "no input")
		os.Exit(2)
	}

	fmt.Fprintln(os.Stdout, "stdout:", scanner.Text())

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(1)
	}
}
