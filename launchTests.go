package main

import (
	"context"
	"flag"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"testk8s/iperf"
	"testk8s/netperf"
	"time"
)

var stars = "*****************************************************************\n"
var labels [2]map[string]string
var nodeptr *apiv1.Node
var nodevect [2]apiv1.Node

func main() {
	clientset := initialSetting()
	//var nod [] v1.Node
	nodes, errNodes := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if errNodes != nil {
		fmt.Println(errNodes)
		return
	}
	count := 1
	var errLabel error
	for i := 0; i < len(nodes.Items); i++ {
		if _, master := nodes.Items[i].Labels["node-role.kubernetes.io/master"]; !master {
			labels[count-1] = nodes.Items[i].GetLabels()
			nodes.Items[i].Labels["type"] = "node" + fmt.Sprintf("%d", count)
			//fmt.Println("label added ")

			nodeptr, errLabel = clientset.CoreV1().Nodes().Update(context.TODO(), &nodes.Items[i], metav1.UpdateOptions{})
			nodevect[count-1] = *nodeptr
			if errLabel != nil {
				fmt.Println(errLabel)
				return
			}
			count++
		}

		if count == 3 {
			break
		}
	}
	fileoutput, err := os.Create(fmt.Sprintf("%d", time.Now().Unix()))
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

	output := iperf.IperfTCPPodtoPod(clientset, 1, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Iperf3 TCP: %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP: " + output + "\n" + stars + "\n")

	output = iperf.IperfUDPPodtoPod(clientset, 1, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Iperf3 UDP: %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP: " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	output = netperf.NetperfTCPPodtoPod(clientset, 1, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Netperf TCP: %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf TCP: " + output + "\n" + stars + "\n")

	output = netperf.NetperfUDPPodtoPod(clientset, 1, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Netperf UDP: %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf UDP: " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	fmt.Println(time.Now())

	fileoutput.WriteString(time.Now().String())

	output = iperf.TCPservice(clientset, 1, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

	output = iperf.UDPservice(clientset, 1, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	output = netperf.TCPservice(clientset, 1, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Netperf TCP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service (1 service in the cluster):: " + output + "\n" + stars + "\n")

	/*output = netperf.UDPservice(clientset, 1, false)
	fmt.Printf("\n%s\navg speed of network Netperf UDP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf UDP with service (1 service in the cluster): " + output + "\n" + stars + "\n")
	*/
	fmt.Println(time.Now())

	fileoutput.WriteString(time.Now().String())

	output = iperf.TCPservice(clientset, 1, true, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

	output = iperf.UDPservice(clientset, 1, true, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	output = netperf.TCPservice(clientset, 1, true, fileoutput)
	fmt.Printf("\n%s\navg speed of network Netperf TCP with service (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = netperf.UDPservice(clientset, 1, true)
		fmt.Printf("\n%s\navg speed of network Netperf UDP with service (multiple services in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Netperf UDP with service (multiple services in the cluster): "+output+"\n"+stars+"\n")*/

	fmt.Println("-----------------------------------------------------------------")
	fileoutput.WriteString("-----------------------------------------------------------------")
	fileoutput.WriteString("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")

	fileoutput.WriteString(time.Now().String())

	fmt.Printf("\n\n")
	fmt.Println("POD TO POD SAME NODE:")
	fileoutput.WriteString("\nPOD TO POD SAME NODE:\n")

	output = iperf.IperfTCPPodtoPod(clientset, 2, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Iperf3 TCP (same node): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP (same node): " + output + "\n" + stars + "\n")

	output = iperf.IperfUDPPodtoPod(clientset, 2, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Iperf3 UDP (same node): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP (same node): " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	output = netperf.NetperfTCPPodtoPod(clientset, 2, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Netperf TCP (same node): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf TCP (same node): " + output + "\n" + stars + "\n")

	output = netperf.NetperfUDPPodtoPod(clientset, 2, fileoutput)
	fmt.Printf("\n%s\navg speed of the network Netperf UDP (same node): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf UDP (same node): " + output + "\n" + stars + "\n")
	fmt.Println(time.Now())

	fileoutput.WriteString(time.Now().String())

	output = iperf.TCPservice(clientset, 2, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

	output = iperf.UDPservice(clientset, 2, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	output = netperf.TCPservice(clientset, 2, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (1 service in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = netperf.UDPservice(clientset, 2, false)
		fmt.Printf("\n%s\navg speed of network Netperf UDP with service(same node) (1 service in the cluster): %s\n %s\n",stars, output,stars)
		fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Netperf UDP with service (1 service in the cluster): "+output+"\n"+stars+"\n")
	*/

	fmt.Println(time.Now())
	fileoutput.WriteString(time.Now().String())

	output = iperf.TCPservice(clientset, 2, true, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")

	output = iperf.UDPservice(clientset, 2, true, fileoutput)
	fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")

	fileoutput.WriteString(time.Now().String())

	output = netperf.TCPservice(clientset, 2, true, fileoutput)
	fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (multiple services in the cluster): %s\n %s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")
	/*

		output = netperf.UDPservice(clientset, 2, true)
			fmt.Printf("\n%s\navg speed of network Netperf UDP with service(same node) (multiple services in the cluster): %s\n %s\n",stars, output,stars)
			fileoutput.WriteString("\n"+stars+"\n"+"avg speed of network Netperf UDP with service (same node) (multiple services in the cluster): "+output+"\n"+stars+"\n")
	*/
	fileoutput.WriteString(time.Now().String())

	fmt.Println(time.Now())

	fileoutput.WriteString("-----------------------------------------------------------------")
	fileoutput.WriteString("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("-----------------------------------------------------------------")

	fmt.Printf("\n\n")
	fmt.Println("HairpinBack:")
	fileoutput.WriteString("\nHairpin back:\n")

	output = netperf.TCPHairpinservice(clientset, false, fileoutput)
	fmt.Printf("\n%s\navg speed of network Netperf TCP Hairpinback with service (1 service in the cluster): %s\n%s\n", stars, output, stars)
	fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP Hairpinback with service (1 service in the cluster): " + output + "\n" + stars + "\n")
	/*
		output = iperf.TCPHairpinservice(clientset, false)
		fmt.Printf("\n%s\navg speed of network iperf TCP Hairpinback with service (1 service in the cluster): %s\n%s\n %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network iperf TCP Hairpinback with service (1 service in the cluster): " + output + "\n" + stars + "\n")
	*/
	fileoutput.WriteString(time.Now().String())

	//todo vedere per network policy
	fmt.Println(time.Now())
	err = fileoutput.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	nodes, errNodes = clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	count = 1
	for i := 0; i < len(nodes.Items); i++ {
		if _, master := nodes.Items[i].Labels["node-role.kubernetes.io/master"]; !master {
			if _, nodeLab := nodes.Items[i].Labels["node-role.kubernetes.io/master"]; !nodeLab {
				nodes.Items[i].SetLabels(labels[count-1])
				delete(nodes.Items[i].Labels, "type")
			}
			_, errLabel = clientset.CoreV1().Nodes().Update(context.TODO(), &nodes.Items[i], metav1.UpdateOptions{})
			if errLabel != nil {
				fmt.Println(errLabel)
				return
			}
			count++
		}

		if count == 3 {
			break
		}
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
