package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Printf("pid=%d\n", os.Getpid())
	fmt.Printf("ppid=%d\n", os.Getppid())
	fmt.Printf("args=%v\n", os.Args)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "no argument")
		os.Exit(2)
	}

	if value, ok := os.LookupEnv("APP_MODE"); ok {
		fmt.Println("APP_MODE=", value)
	} else {
		fmt.Println("APP_MODE is not set")
	}

	fmt.Fprintln(os.Stdout, "stdout: ", os.Args[1])

	// システムの停止や終了のシグナルを受け取るためのチャネルを作る
	sigCh := make(chan os.Signal, 1)
	// 定義したチャネルには、os.Interrupt（Ctrl+C）やsyscall.SIGTERM（終了シグナル）を通知するように設定する
	// signal.Notify()によってシグナルが送信されるとsigChチャネルに通知される
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	// プログラム終了時にsignal.Notify()の設定を解除するためにdeferでsignal.Stop()を呼び出す
	defer signal.Stop(sigCh)

	fmt.Println("type one line and press Enter:")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "no input")
		os.Exit(2)
	}

	fmt.Fprintln(os.Stdout, "stdout:", scanner.Text())
	fmt.Println("waiting for Ctrl+C or SIGTERM")

	// チャネルがシグナルを受信するまでブロックする
	sig := <-sigCh
	// シグナルを受信したら受信したシグナルの内容を出力する
	fmt.Println("received signal:", sig)
}
