package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"testk8s/iperf"
	"time"
)

var stars = "*****************************************************************\n"

func main() {
	clientset := initialSetting()
	fileoutput, err := os.Create(time.Now().String())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(time.Now())
	fileoutput.WriteString(time.Now().String())
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("\n\n")
	fileoutput.WriteString("\nPOD TO POD DIFFERENT NODES:\n")
	fmt.Println("POD TO POD DIFFERENT NODES:")

	output := iperf.IperfTCPPodtoPod(clientset, 1)
	fmt.Printf("\n%s\navg speed of the network Iperf3 TCP: %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP: " + output + "\n" + stars + "\n")

	output = iperf.IperfUDPPodtoPod(clientset, 1)
	fmt.Printf("\n%s\navg speed of the network Iperf3 UDP: %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP: " + output + "\n" + stars + "\n")

	/*
		output = netperf.NetperfTCPPodtoPod(clientset, 1)
		fmt.Printf("\n%s\navg speed of the network Netperf TCP: %s\n %s\n",stars,output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of the network Netperf TCP: "+output+"\n"+stars+"\n")

		output = netperf.NetperfUDPPodtoPod(clientset, 1)
		fmt.Printf("\n%s\navg speed of the network Netperf UDP: %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of the network Netperf UDP: "+output+"\n"+stars+"\n")
	*/

	fmt.Println(time.Now())

	output = iperf.TCPservice(clientset, 1, false)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = iperf.UDPservice(clientset, 1, false)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (1 service in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.TCPservice(clientset, 1, false)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 TCP with service (1 service in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.UDPservice(clientset, 1, false)
		fmt.Printf("\n%s\navg speed of network Netperf UDP with service (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (1 service in the cluster): "+output+"\n"+stars+"\n")
	*/

	fmt.Println(time.Now())

	output = iperf.TCPservice(clientset, 1, true)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = iperf.UDPservice(clientset, 1, true)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (multiple services in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.TCPservice(clientset, 1, true)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 TCP with service (multiple services in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.UDPservice(clientset, 1, true)
		fmt.Printf("\n%s\navg speed of network Netperf UDP with service (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (multiple services in the cluster): "+output+"\n"+stars+"\n")
	*/
	fmt.Println("-----------------------------------------------------------------")
	fileoutput.WriteString("-----------------------------------------------------------------")
	fileoutput.WriteString("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")

	fmt.Printf("\n\n")
	fmt.Println("POD TO POD SAME NODE:")
	fileoutput.WriteString("\nPOD TO POD SAME NODE:\n")

	output = iperf.IperfTCPPodtoPod(clientset, 2)
	fmt.Printf("\n%s\navg speed of the network Iperf3 TCP (same node): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP (same node): " + output + "\n" + stars + "\n")

	output = iperf.IperfUDPPodtoPod(clientset, 2)
	fmt.Printf("\n%s\navg speed of the network Iperf3 UDP (same node): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP (same node): " + output + "\n" + stars + "\n")
	/*
		output = netperf.NetperfTCPPodtoPod(clientset, 2)
		fmt.Printf("\n%s\navg speed of the network Netperf TCP (same node): %s\n %s\n",stars,output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of the network Netperf TCP (same node): "+output+"\n"+stars+"\n")

		output = netperf.NetperfUDPPodtoPod(clientset, 2)
		fmt.Printf("\n%s\navg speed of the network Netperf UDP (same node): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of the network Netperf UDP (same node): "+output+"\n"+stars+"\n")

	*/
	fmt.Println(time.Now())

	output = iperf.TCPservice(clientset, 2, false)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = iperf.UDPservice(clientset, 2, false)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (1 service in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.TCPservice(clientset, 2, false)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 TCP with service (1 service in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.UDPservice(clientset, 2, false)
		fmt.Printf("\n%s\navg speed of network Netperf UDP with service(same node) (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (1 service in the cluster): "+output+"\n"+stars+"\n")

	*/
	fmt.Println(time.Now())

	output = iperf.TCPservice(clientset, 2, true)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = iperf.UDPservice(clientset, 2, true)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (same node) (multiple services in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.TCPservice(clientset, 2, true)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 TCP with service (same node) (multiple services in the cluster): "+output+"\n"+stars+"\n")

		output = netperf.UDPservice(clientset, 2, true)
		fmt.Printf("\n%s\navg speed of network Netperf UDP with service(same node) (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Iperf3 UDP with service (same node) (multiple services in the cluster): "+output+"\n"+stars+"\n")
	*/

	fmt.Println(time.Now())

	fileoutput.WriteString("-----------------------------------------------------------------")
	fileoutput.WriteString("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")

	fmt.Printf("\n\n")
	fmt.Println("HairpinBack:")
	fileoutput.WriteString("\nHairpin back:\n")
	/*fmt.Printf("\n%s\navg speed of network Netperf TCP with service (1 service in the cluster): %s\n%s\n",stars, netperf.TCPHairpinservice(clientset,false),stars)
	fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (1 service in the cluster): %s\n %s\n",stars, iperfPTP.UDPHairpinservice(clientset,false),stars)
	fmt.Printf("\n%s\navg speed of network Netperf UDP with service (1 service in the cluster): %s\n %s\n", stars, netperf.UDPHairpinservice(clientset, false), stars)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (multiple services in the cluster): %s\n%s\n", stars, iperfPTP.TCPHairpinservice(clientset, true), stars)
	fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (multiple services in the cluster): %s\n%s\n", stars, iperfPTP.UDPHairpinservice(clientset, true), stars)*/
	//fmt.Printf("\n%s\navg speed of network Netperf TCP with service (multiple services in the cluster): %s\n%s\n", stars, netperf.TCPHairpinservice(clientset, true), stars)
	//fmt.Printf("\n%s\navg speed of network Netperf UDP with service (multiple services in the cluster): %s\n%s\n", stars, netperf.UDPHairpinservice(clientset, true), stars)*/

	//todo vedere per network policy

	err = fileoutput.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
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
