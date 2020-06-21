package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"testk8s/netperf"
	"time"
)

func main() {
	clientset := initialSetting()
	fmt.Println(time.Now())
	fmt.Println("POD TO POD DIFFERENT NODE")
	/*fmt.Printf("avg speed of the network Iperf3 TCP: %s\n", iperfPTP.IperfTCPPodtoPod(clientset, 1))
	fmt.Println(time.Now())
	fmt.Printf("avg speed of the network Netperf TCP: %s\n", netperfPTP.NetperfTCPPodtoPod(clientset, 1))
	fmt.Println(time.Now())
	fmt.Printf("avg speed of the network Iperf3 UDP: %s\n", iperfPTP.IperfUDPPodtoPod(clientset, 1))
	fmt.Println(time.Now())
	fmt.Printf("avg speed of the network Netperf UDP: %s\n", netperfPTP.NetperfUDPPodtoPod(clientset, 1))
	fmt.Println(time.Now())*/
	//fmt.Printf("avg speed of network Iperf3 TCP with service: %s\n", iperfPTP.TCPservice(clientset, 1))
	//fmt.Printf("avg speed of network Iperf3 TCP with service: %s\n", iperfPTP.UDPservice(clientset, 1))
	fmt.Printf("avg speed of network Netperf TCP with service: %s\n", netperf.TCPservice(clientset, 1))

	fmt.Println("POD TO POD SAME NODE")
}

func initialSetting() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
