package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <host>", os.Args[0])
	}

	host := os.Args[1]
	ctx := context.Background()

	// OSが持っている問い合わせ先DNS設定などを参照してDNSにドメイン情報を問い合わせるためのリゾルバを生成
	resolver := net.Resolver{}

	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		log.Fatal(err)
	}

	hosts, err := resolver.LookupHost(ctx, host)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("LookupHost:")
	for _, host := range hosts {
		fmt.Println(" ", host)
	}

	fmt.Println("name:", host)
	fmt.Println("A:")
	for _, ip := range ips {
		if ip.To4() != nil {
			fmt.Println(" ", ip.String())
		}
	}

	fmt.Println("AAAA:")
	for _, ip := range ips {
		if ip.To4() == nil {
			fmt.Println(" ", ip.String())
		}
	}

	// 引数のホスト名からCNAME情報を取得する
	// CNAMEとはそのホストにつけられたエイリアス情報
	fmt.Println("CNAME:")
	cname, err := resolver.LookupCNAME(ctx, host)
	if err != nil {
		fmt.Println(" <lookup error>", err)
	} else {
		fmt.Println(" ", cname)
	}

	// MXはそのドメイン宛のメール配送先となるメールサーバー情報
	fmt.Println("MX:")
	mxs, err := resolver.LookupMX(ctx, host)
	if err != nil {
		fmt.Println(" <lookup error>", err)
		return
	}
	if len(mxs) == 0 {
		fmt.Println(" <none>")
		return
	}

	for _, mx := range mxs {
		fmt.Printf("  %s priority=%d\n", mx.Host, mx.Pref)
	}
}
