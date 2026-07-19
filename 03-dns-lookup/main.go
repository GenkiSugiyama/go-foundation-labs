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
	resolver := net.Resolver{}

	ips, err := resolver.LookupIP(context.Background(), "ip", host)
	if err != nil {
		log.Fatal(err)
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
}
