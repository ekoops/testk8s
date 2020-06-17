package main

import (
	"fmt"
	iperfPTP "testk8s/iperf"
)

func main() {
	fmt.Println("File da cui lanciare tutto")
	fmt.Println("Pod to Pod test with Iperf3: ")
	//fmt.Printf("avg speed of the network Iperf3 TCP: %s\n", iperfPTP.IperfTCPPodtoPod())
	fmt.Printf("avg speed of the network Iperf3 UDP: %s\n", iperfPTP.IperfUDPPodtoPod())
}
