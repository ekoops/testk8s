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
	"strconv"
	"testk8s/curl"
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
	netPolRep := [4]int{10, 20, 50, 100}
	netPolServices := [4]int{1, 100, 1000, 10000}
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
	/*
		fmt.Println(time.Now())
		fileoutput.WriteString(time.Now().String())
		fmt.Println("-----------------------------------------------------------------")
		fmt.Println("-----------------------------------------------------------------")
		fmt.Printf("\n\n")
		fileoutput.WriteString("\nPOD TO POD DIFFERENT NODES:\n")
		fmt.Println("POD TO POD DIFFERENT NODES:")

		output := iperf.IperfTCPPodtoPod(clientset, 1, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Iperf3 TCP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP: " + output + "\n" + stars + "\n")

		output = iperf.IperfUDPPodtoPod(clientset, 1, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Iperf3 UDP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP: " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.NetperfTCPPodtoPod(clientset, 1, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Netperf TCP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf TCP: " + output + "\n" + stars + "\n")

		output = netperf.NetperfUDPPodtoPod(clientset, 1, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Netperf UDP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf UDP: " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		fmt.Println(time.Now())

		fileoutput.WriteString(time.Now().String())

		output = iperf.TCPservice(clientset, 1, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.UDPservice(clientset, 1, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 1, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service (1 service in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service (1 service in the cluster):: " + output + "\n" + stars + "\n")

		fmt.Println(time.Now())

		fileoutput.WriteString(time.Now().String())
		output = iperf.TCPservice(clientset, 1, true, fileoutput, 10)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (10 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.UDPservice(clientset, 1, true, fileoutput, 10)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (10 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 1, true, fileoutput, 10)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service (10 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.TCPservice(clientset, 1, true, fileoutput, 10000)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (10000 services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.UDPservice(clientset, 1, true, fileoutput, 10000)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service (10000 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 1, true, fileoutput, 10000)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service (10000 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service (multiple services in the cluster): " + output + "\n" + stars + "\n")

		fmt.Println("-----------------------------------------------------------------")
		fileoutput.WriteString("-----------------------------------------------------------------")
		fileoutput.WriteString("-----------------------------------------------------------------")
		fmt.Println("-----------------------------------------------------------------")

		fileoutput.WriteString(time.Now().String())

		fmt.Printf("\n\n")
		fmt.Println("POD TO POD SAME NODE:")
		fileoutput.WriteString("\nPOD TO POD SAME NODE:\n")
		output = iperf.IperfTCPPodtoPod(clientset, 2, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Iperf3 TCP (same node): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP (same node): " + output + "\n" + stars + "\n")

		output = iperf.IperfUDPPodtoPod(clientset, 2, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Iperf3 UDP (same node): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP (same node): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.NetperfTCPPodtoPod(clientset, 2, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Netperf TCP (same node): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf TCP (same node): " + output + "\n" + stars + "\n")

		output = netperf.NetperfUDPPodtoPod(clientset, 2, fileoutput, false, 0)
		fmt.Printf("\n%s\navg speed of the network Netperf UDP (same node): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf UDP (same node): " + output + "\n" + stars + "\n")
		fmt.Println(time.Now())

		fileoutput.WriteString(time.Now().String())

		output = iperf.TCPservice(clientset, 2, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.UDPservice(clientset, 2, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 2, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (1 service in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (1 service in the cluster): " + output + "\n" + stars + "\n")

		fmt.Println(time.Now())
		fileoutput.WriteString(time.Now().String())

		output = iperf.TCPservice(clientset, 2, true, fileoutput, 10)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (10 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.UDPservice(clientset, 2, true, fileoutput, 10)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (10 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 2, true, fileoutput, 10)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (10 multiple services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (multiple services in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.TCPservice(clientset, 2, true, fileoutput, 10000)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (10000 services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

		output = iperf.UDPservice(clientset, 2, true, fileoutput, 10000)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(same node) (10000 services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (1 service in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 2, true, fileoutput, 10000)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (10000 services in the cluster): %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (1 service in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		fmt.Println(time.Now())

		fileoutput.WriteString("-----------------------------------------------------------------")
		fileoutput.WriteString("-----------------------------------------------------------------")
		fmt.Println("-----------------------------------------------------------------")
		fmt.Println("-----------------------------------------------------------------")

		fmt.Printf("\n\n")
		fmt.Println("HairpinBack:")
		fileoutput.WriteString("\nHairpin back:\n")

		output = netperf.TCPHairpinservice(clientset, false, fileoutput, 1)
		fmt.Printf("\n%s\navg speed of network Netperf TCP Hairpinback with service (1 service in the cluster): %s\n%s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP Hairpinback with service (1 service in the cluster): " + output + "\n" + stars + "\n")
		fileoutput.WriteString(time.Now().String())

		// Parte aggiuntiva di curl
	*/
	output := "ciao"
	for i := 1; i < 4; i++ {
		for j := 0; j < 1; j++ {
			output = curl.SpeedMovingFileandLatency(clientset, netPolRep[i], 1, fileoutput, netPolServices[j])
			fmt.Printf("\n%s\n Network speed and latency with a growing number of services and endpoints: %s\n%s\n", stars, output, stars)
			fileoutput.WriteString("\n" + stars + "\n" + "Network speed and latency with a growing number of services " + strconv.Itoa(netPolServices[i]) + " and endpoints " + strconv.Itoa(netPolRep[i]) + " : " + output + "\n" + stars + "\n")
		}
	}

	// parte aggiuntiva su test con molti pods nel cluster

	multiple := false
	numServ := 1
	for i := 0; i < 5; i++ {
		output = iperf.TCPservice(clientset, 1, multiple, fileoutput, numServ)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service (%d service in the cluster): %s\n %s\n", stars, numServ, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (" + strconv.Itoa(numServ) + " service in the cluster): " + output + "\n" + stars + "\n")
		output = iperf.TCPservice(clientset, 2, multiple, fileoutput, numServ)
		fmt.Printf("\n%s\navg speed of network Iperf3 TCP with service(same node) (%d service in the cluster): %s\n %s\n", stars, numServ, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 TCP with service (" + strconv.Itoa(numServ) + " service in the cluster): " + output + "\n" + stars + "\n")
		multiple = true
		numServ = numServ * 10
	}

	multiple = false
	numServ = 100
	for i := 0; i < 3; i++ {

		output = iperf.UDPservice(clientset, 1, multiple, fileoutput, numServ)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(%d service in the cluster): %s\n %s\n", stars, numServ, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (" + strconv.Itoa(numServ) + " service in the cluster): " + output + "\n" + stars + "\n")
		output = iperf.UDPservice(clientset, 2, multiple, fileoutput, numServ)
		fmt.Printf("\n%s\navg speed of network Iperf3 UDP with service(%d service in the cluster): %s\n %s\n", stars, numServ, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Iperf3 UDP with service (" + strconv.Itoa(numServ) + " service in the cluster): " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.TCPservice(clientset, 1, multiple, fileoutput, numServ)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(%d service in the cluster): %s\n %s\n", stars, numServ, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (" + strconv.Itoa(numServ) + " service in the cluster): " + output + "\n" + stars + "\n")
		output = netperf.TCPservice(clientset, 2, multiple, fileoutput, numServ)
		fmt.Printf("\n%s\navg speed of network Netperf TCP with service(same node) (%d service in the cluster): %s\n %s\n", stars, numServ, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of network Netperf TCP with service(same node) (" + strconv.Itoa(numServ) + " service in the cluster): " + output + "\n" + stars + "\n")

		numServ = numServ * 10
		multiple = true
	}

	//parte con netpol installate nel cluster
	fmt.Println("Network Policies Part")
	for i := 0; i < 2; i++ {
		output = iperf.IperfTCPPodtoPod(clientset, 1, fileoutput, true, 1000*i+1)
		fmt.Printf("\n%s\navg speed of the network Iperf3 TCP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 TCP: " + output + "\n" + stars + "\n")

		output = iperf.IperfUDPPodtoPod(clientset, 1, fileoutput, true, 1000*i+1)
		fmt.Printf("\n%s\navg speed of the network Iperf3 UDP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Iperf3 UDP: " + output + "\n" + stars + "\n")

		fileoutput.WriteString(time.Now().String())

		output = netperf.NetperfTCPPodtoPod(clientset, 1, fileoutput, true, 1000*i+1)
		fmt.Printf("\n%s\navg speed of the network Netperf TCP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf TCP: " + output + "\n" + stars + "\n")

		output = netperf.NetperfUDPPodtoPod(clientset, 1, fileoutput, true, 1000*i+1)
		fmt.Printf("\n%s\navg speed of the network Netperf UDP: %s\n %s\n", stars, output, stars)
		fileoutput.WriteString("\n" + stars + "\n" + "avg speed of the network Netperf UDP: " + output + "\n" + stars + "\n")

	}

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
