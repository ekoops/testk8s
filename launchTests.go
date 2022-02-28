package main

import (
	"context"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"testk8s/curl"
	"testk8s/iperf"
	"time"
)

var stars = "*****************************************************************\n"

func addTypeLabel(cset *kubernetes.Clientset) error {
	nodes, err := cset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := 0; i < len(nodes.Items); i++ {
		nodes.Items[i].Labels["type"] = "node" + fmt.Sprintf("%d", i+1)
		_, err := cset.CoreV1().Nodes().Update(context.TODO(), &nodes.Items[i], metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func removeTypeLabel(cset *kubernetes.Clientset) error {
	nodes, err := cset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := 0; i < len(nodes.Items); i++ {
		delete(nodes.Items[i].Labels, "type")
		_, err = cset.CoreV1().Nodes().Update(context.TODO(), &nodes.Items[i], metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	}
	return nil
}

func printNow(f *os.File) {
	now := time.Now().String()
	fmt.Println(now)
	f.WriteString(now)
}

func testPodToPod(cset *kubernetes.Clientset, diffNodes bool, f *os.File) {
	printNow(f)

	output := iperf.IperfTCPPodtoPod(cset, diffNodes, f, false, 0)
	s := fmt.Sprintf("\n%s\navg speed of the network Iperf3 TCP (diff. nodes: %t): %s\n %s\n", stars, diffNodes, output, stars)
	fmt.Print(s)
	f.WriteString(s)

	output = iperf.IperfUDPPodtoPod(cset, diffNodes, f, false, 0)
	s = fmt.Sprintf("\n%s\navg speed of the network Iperf3 UDP (diff. nodes: %t): %s\n %s\n", stars, diffNodes, output, stars)
	fmt.Print(s)
	f.WriteString(s)

	f.WriteString(time.Now().String())

	//output = netperf.NetperfTCPPodtoPod(cset, diffNodes, f, false, 0)
	//s = fmt.Sprintf("\n%s\navg speed of the network Netperf TCP (diff. nodes: %t): %s\n %s\n", stars, diffNodes, output, stars)
	//fmt.Print(s)
	//f.WriteString(s)
	//
	//output = netperf.NetperfUDPPodtoPod(cset, diffNodes, f, false, 0)
	//s = fmt.Sprintf("\n%s\navg speed of the network Netperf UDP (diff. nodes: %t): %s\n %s\n", stars, diffNodes, output, stars)
	//fmt.Print(s)
	//f.WriteString(s)

	printNow(f)
}

func testPodToSvc(cset *kubernetes.Clientset, diffNodes bool, multiple bool, svcNum int, f *os.File) {
	printNow(f)

	output := iperf.TCPservice(cset, diffNodes, multiple, f, svcNum)
	s := fmt.Sprintf(
		"\n%s\navg speed of network Iperf3 TCP with service (diff. nodes: %t; services in cluster: %d): %s\n %s\n",
		stars, diffNodes, svcNum, output, stars,
	)
	fmt.Print(s)
	f.WriteString(s)

	output = iperf.UDPservice(cset, diffNodes, multiple, f, svcNum)
	s = fmt.Sprintf(
		"\n%s\navg speed of network Iperf3 UDP with service (diff. nodes: %t; services in cluster: %d): %s\n %s\n",
		stars, diffNodes, svcNum, output, stars,
	)
	fmt.Print(s)
	f.WriteString(s)

	f.WriteString(time.Now().String())

	//output = netperf.TCPservice(cset, diffNodes, multiple, f, svcNum)
	//s = fmt.Sprintf(
	//	"\n%s\navg speed of network Netperf TCP with service (diff. nodes: %t; services in cluster: %d): %s\n %s\n",
	//	stars, diffNodes, svcNum, output, stars,
	//)
	//fmt.Print(s)
	//f.WriteString(s)

	printNow(f)
}

func main() {
	clientset := initialSetting()

	if err := addTypeLabel(clientset); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fileoutput, err := os.Create(fmt.Sprintf("%d", time.Now().Unix()))
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return
	}

	now := time.Now().String()
	fmt.Printf("%s\n-----------------------------------------------------------------\n\n\n", now)
	fmt.Println("POD TO POD DIFFERENT NODES:")
	fileoutput.WriteString(fmt.Sprintf("%s\nPOD TO POD DIFFERENT NODES:\n", now))

	testPodToPod(clientset, true, fileoutput)

	testPodToSvc(clientset, true, false, 1, fileoutput)

	testPodToSvc(clientset, true, true, 10, fileoutput)

	now = time.Now().String()
	fmt.Printf("%s\n-----------------------------------------------------------------\n\n\n", now)
	fmt.Println("POD TO POD SAME NODE:")
	fileoutput.WriteString(fmt.Sprintf("%s\nPOD TO POD SAME NODE:\n", now))

	testPodToPod(clientset, false, fileoutput)

	testPodToSvc(clientset, false, false, 1, fileoutput)

	testPodToSvc(clientset, false, true, 10, fileoutput)

	// Parte aggiuntiva di curl
	netPolRep := [4]int{10, 20, 50, 100}
	netPolServices := [4]int{1, 100, 1000, 10000}
	for i := 0; i < 4; i++ {
		for j := 3; j < 4; j++ {
			output := curl.SpeedMovingFileandLatency(clientset, netPolRep[i], true, fileoutput, netPolServices[j])
			s := fmt.Sprintf(
				"\n%s\nNetwork speed and latency with a growing number of services %d and endpoints %d : %s\n%s\n",
				stars, netPolServices[j], netPolRep[i], output, stars,
			)
			fmt.Print(s)
			fileoutput.WriteString(s)
		}
	}

	fmt.Println(time.Now())
	if err := fileoutput.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := removeTypeLabel(clientset); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
