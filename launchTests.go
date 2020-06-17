package main

import (
	"fmt"
	iperfPTP "testk8s/iperf"
)

func main() {
	fmt.Println("File da cui lanciare tutto")
	fmt.Println("Pod to Pod test with Iperf3: ")
	fmt.Printf("avg speed of the network: %s\n", iperfPTP.IperfPodtoPod())
}
