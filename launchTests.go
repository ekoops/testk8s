package main

import (
	"fmt"
	iperfPTP "testk8s/iperf"
)

func main() {
	fmt.Println("File da cui lanciare tutto")
	fmt.Println("Pod to Pod test with Iperf3: ")
	fmt.Printf("Test 1: %s\n", iperfPTP.IperfPodtoPod())
}
