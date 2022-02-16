package iperf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"os"
	"strconv"
	"strings"
	"testk8s/utils"
)

var nameService = "my-service-iperf"
var namespaceUDP = "testiperfudp"
var imageServiceUDP = "leannet/k8s-netperf:latest"

func TCPservice(clientset *kubernetes.Clientset, casus bool, multiple bool, fileoutput *os.File, numberServices int) string {

	node = utils.SetNodeSelector(casus)
	nsCR := utils.CreateNS(clientset, namespace)
	fmt.Printf("Namespace %s created \n", nsCR.Name)

	netSpeeds := make([]float64, iteration)
	cpuServ := make([]float64, iteration)
	cpuClie := make([]float64, iteration)
	cpuconfC := make([]float64, iteration)
	cpuconfS := make([]float64, iteration)

	if multiple {
		fmt.Println("the program will create multiple service and endpoints")
		utils.CreateBulk(numberServices, numberServices, clientset, namespace)
	}
	svcCr := createTCPService(clientset, "iperfserver")

	part := 0
	for i := 0; i <= (iteration / 4); i++ {
		commandD := "iperf3 -s -p 5001 -V"
		dep := createIperfDeployment(namespace, image, commandD)
		fmt.Println("Creating deployment...")
		res, errDepl := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), dep, metav1.CreateOptions{})
		if errDepl != nil {
			panic(errDepl)
		}
		fmt.Printf("Created deployment %q.\n", res.GetObjectMeta().GetName())
		deps, errD := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if errD != nil {
			panic(errD.Error())
		}

		podvect, errP := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfserver"})
		if errP != nil {
			panic(errP)
		}
		fmt.Print("Wait for pod creation..")
		for {
			podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfserver"})
			if errP != nil {
				panic(errP)
			}
			var num int = len(podvect.Items)
			if num != 0 {
				fmt.Printf("\n")
				break
			}
			fmt.Print(".")
		}
		fmt.Printf("There are %d pods and %d depl in the cluster\n", len(podvect.Items), len(deps.Items))
		pod := podvect.Items[0]
		ctl := 0
		for ctl != 1 {
			switch pod.Status.Phase {
			case apiv1.PodRunning:
				{
					ctl = 1
					break
				}
			case apiv1.PodPending:
				{
					podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfserver"})
					if errP != nil {
						panic(errP)
					}
					pod = podvect.Items[0]
				}
			case apiv1.PodFailed, apiv1.PodSucceeded:
				panic("error in pod creation")
			}
		}
		serviceC, errSvcSearch := clientset.CoreV1().Services(namespace).Get(context.TODO(), nameService, metav1.GetOptions{})
		if errors.IsNotFound(errSvcSearch) {
			fmt.Printf("svc %s in namespace %s not found\n", nameService, namespace)
		} else if statusError, isStatus := errSvcSearch.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting svc %s in namespace %s: %v\n",
				nameService, namespace, statusError.ErrStatus.Message)
		} else if errSvcSearch != nil {
			panic(errSvcSearch.Error())
		} else {
			fmt.Printf("Found svc %s in namespace %s\n", svcCr.Name, namespace)
			svcIP := serviceC.Spec.ClusterIP
			fmt.Printf("Service IP: %s\n", svcIP)
		}

		command := "for i in 0 1 2; do iperf3 -c " + serviceC.Spec.ClusterIP + " -p 5001 -V -N -t 10 -Z -A 1,2 -M 1448 >> file.txt; sleep 11; done;cat file.txt"
		fmt.Println("Creating Iperf Client: " + command)
		jobsClient := clientset.BatchV1().Jobs(namespace)
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit: pointer.Int32Ptr(4),
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "iperfclient",
						Labels: map[string]string{"app": "iperfclient"},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:    "iperfclient",
								Image:   image,
								Command: []string{"/bin/bash"},
								Args:    []string{"-c", command},
							},
						},
						RestartPolicy: "OnFailure",
						NodeSelector:  map[string]string{"type": node},
					},
				},
			},
		}
		result1, errJ := jobsClient.Create(context.TODO(), job, metav1.CreateOptions{})
		if errJ != nil {
			fmt.Println(errJ.Error())
			panic(errJ)
		}
		fmt.Printf("Created job %q.\n", result1.GetObjectMeta().GetName())
		podClient, errC := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfclient"})
		if errC != nil {
			panic(errC)
		}
		for {
			if len(podClient.Items) != 0 {
				break
			}
			podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfclient"})
			if errC != nil {
				panic(errC)
			}
		}
		fmt.Printf("Created pod %q.\n", podClient.Items[0].Name)
		pod = podClient.Items[0]
		var str string
		ctl = 0
		for ctl != 1 {
			switch pod.Status.Phase {
			case apiv1.PodRunning, apiv1.PodPending:
				{
					podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfclient"})
					if errC != nil {
						panic(errC)
					}
					pod = podClient.Items[0]
				}
			case apiv1.PodSucceeded:
				{
					logs := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &apiv1.PodLogOptions{})
					podLogs, errLogs := logs.Stream(context.TODO())
					if errLogs != nil {
						panic(errLogs)
					}
					defer podLogs.Close()
					buf := new(bytes.Buffer)
					_, errBuf := io.Copy(buf, podLogs)
					if errBuf != nil {
						panic(errBuf)
					}
					str = buf.String()
					fileoutput.WriteString(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		//todo stampare su file e non a video
		fmt.Println(str)
		if strings.Contains(str, "error - unable") {
			i = i - 1
			utils.CleanCluster(clientset, namespace, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
			continue
		}
		velspeed, cpuClient, cpuServer := parseVel(str, clientset, namespaceUDP)
		fmt.Printf("%d %f Gbits/sec 	clientcpu: %f	server cpu :%f\n", i, velspeed, cpuClient, cpuServer)
		//todo vedere cosa succede con float 32, per ora 64
		for j := 0; j < 3; j++ {
			netSpeeds[part] = velspeed[j]
			cpuClie[part] = cpuClient[j]
			cpuServ[part] = cpuServer[j]
			part++
		}

		//Clean the cluster
		utils.CleanCluster(clientset, namespace, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)

	}

	utils.DeleteBulk(numberServices, numberServices, clientset, namespace)
	utils.DeleteNS(clientset, namespace)
	fmt.Printf("Namespace %s deleted \n", namespace)
	avgS, avgCPUC, avgCPUS, _, _ := utils.AvgSpeed(netSpeeds, cpuClie, cpuServ, cpuconfC, cpuconfS, float64(iteration))
	return fmt.Sprintf("%f", avgS) + " Gbits/sec, client cpu usage: " + fmt.Sprintf("%f", avgCPUC) + " and server CPU usage: " + fmt.Sprintf("%f", avgCPUS)

}

func UDPservice(clientset *kubernetes.Clientset, casus bool, multiple bool, fileoutput *os.File, numberServices int) string {

	node = utils.SetNodeSelector(casus)
	nsCR := utils.CreateNS(clientset, namespaceUDP)
	fmt.Printf("Namespace %s created \n", nsCR.GetName())
	netSpeeds := make([]float64, iteration)
	cpuClie := make([]float64, iteration)
	cpuClie[0] = -100.00
	cpuServ := make([]float64, iteration)

	if multiple {
		utils.CreateBulk(numberServices, numberServices, clientset, namespaceUDP)
	}

	svc := apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService,
			Namespace: namespaceUDP,
			//Labels: map[string]string{"":""},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Protocol:   apiv1.ProtocolUDP,
				Port:       5003,
				TargetPort: intstr.FromInt(5003),
			}},
			Selector: map[string]string{"app": "iperfserver"},
		},
	}
	svcCr, errCr := clientset.CoreV1().Services(namespaceUDP).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if errCr != nil {
		panic(errCr)
	}
	fmt.Println("Service UDP my-service-iperf created " + svcCr.GetName())

	part := 0
	for i := 0; i <= (iteration / 4); i++ {
		commandD := "iperf -s -u -p 5003"
		dep := createIperfDeployment(namespaceUDP, imageServiceUDP, commandD)
		fmt.Println("Creating deployment...")
		res, errDepl := clientset.AppsV1().Deployments(namespaceUDP).Create(context.TODO(), dep, metav1.CreateOptions{})
		if errDepl != nil {
			panic(errDepl)
		}
		fmt.Printf("Created deployment %q.\n", res.GetObjectMeta().GetName())
		deps, errD := clientset.AppsV1().Deployments(namespaceUDP).List(context.TODO(), metav1.ListOptions{})
		if errD != nil {
			panic(errD.Error())
		}

		podvect, errP := clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{})
		if errP != nil {
			panic(errP)
		}
		fmt.Print("Wait for pod creation..")
		for {
			podvect, errP = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{})
			if errP != nil {
				panic(errP)
			}
			var num int = len(podvect.Items)
			if num != 0 {
				fmt.Printf("\n")
				break
			}
			fmt.Print(".")
		}
		fmt.Printf("There are %d pods and %d depl in the cluster\n", len(podvect.Items), len(deps.Items))
		pod := podvect.Items[0]
		ctl := 0
		for ctl != 1 {
			switch pod.Status.Phase {
			case apiv1.PodRunning:
				{
					ctl = 1
					break
				}
			case apiv1.PodPending:
				{
					podvect, errP = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{})
					if errP != nil {
						panic(errP)
					}
					pod = podvect.Items[0]
				}
			case apiv1.PodFailed, apiv1.PodSucceeded:
				panic("error in pod creation")
			}
		}
		serviceC, errSvcSearch := clientset.CoreV1().Services(namespaceUDP).Get(context.TODO(), nameService, metav1.GetOptions{})
		if errors.IsNotFound(errSvcSearch) {
			fmt.Printf("svc %s in namespace %s not found\n", nameService, namespaceUDP)
		} else if statusError, isStatus := errSvcSearch.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting svc %s in namespace %s: %v\n",
				nameService, namespaceUDP, statusError.ErrStatus.Message)
		} else if errSvcSearch != nil {
			panic(errSvcSearch.Error())
		} else {
			fmt.Printf("Found svc %s in namespace %s\n", svc.Name, namespaceUDP)
			svcIP := serviceC.Spec.ClusterIP
			fmt.Printf("Service IP: %s\n", svcIP)
		}

		command := "for i in 0 1 2; do iperf -c " + serviceC.Spec.ClusterIP + " -u -b 10000G -p 5003 -V -i 1 -t 10 >> file.txt; sleep 11 ;done; cat file.txt"
		fmt.Println("Creating UDP Iperf Client: " + command)
		jobsClient := clientset.BatchV1().Jobs(namespaceUDP)
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespaceUDP,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit: pointer.Int32Ptr(4),
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "iperfclient",
						Labels: map[string]string{"app": "iperfclient"},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:    "iperfclient",
								Image:   imageServiceUDP,
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
							},
						},
						RestartPolicy: "OnFailure",
						NodeSelector:  map[string]string{"type": node},
					},
				},
			},
		}
		result1, errJ := jobsClient.Create(context.TODO(), job, metav1.CreateOptions{})
		if errJ != nil {
			fmt.Println(errJ.Error())
			panic(errJ)
		}
		fmt.Printf("Created job %q.\n", result1.GetObjectMeta().GetName())
		podClient, errC := clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfclient"})
		if errC != nil {
			panic(errC)
		}
		for {
			if len(podClient.Items) != 0 {
				break
			}
			podClient, errC = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfclient"})
			if errC != nil {
				panic(errC)
			}
		}
		fmt.Printf("Created pod %q.\n", podClient.Items[0].Name)
		pod = podClient.Items[0]
		var str string
		ctl = 0
		for ctl != 1 {
			switch pod.Status.Phase {
			case apiv1.PodRunning, apiv1.PodPending:
				{
					podClient, errC = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=iperfclient"})
					if errC != nil {
						panic(errC)
					}
					pod = podClient.Items[0]
				}
			case apiv1.PodSucceeded:
				{
					logs := clientset.CoreV1().Pods(namespaceUDP).GetLogs(pod.Name, &apiv1.PodLogOptions{})
					podLogs, errLogs := logs.Stream(context.TODO())
					if errLogs != nil {
						panic(errLogs)
					}
					defer podLogs.Close()
					buf := new(bytes.Buffer)
					_, errBuf := io.Copy(buf, podLogs)
					if errBuf != nil {
						panic(errBuf)
					}
					str = buf.String()
					fmt.Println(str)
					fileoutput.WriteString(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		if strings.Contains(str, "write failed:") || strings.Contains(str, "read failed:") {
			i = i - 1
			utils.CleanCluster(clientset, namespaceUDP, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
			continue
		}
		//works on strings
		velspeed := parseVelServiceUdp(str, clientset, namespaceUDP)
		fmt.Printf("%d %f Gbits/sec\n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		for j := 0; j < 3; j++ {
			netSpeeds[part] = velspeed[j]
			part++
		}
		utils.CleanCluster(clientset, namespaceUDP, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
	}
	if multiple {
		utils.DeleteBulk(numberServices, numberServices, clientset, namespaceUDP)
	}
	utils.DeleteNS(clientset, namespaceUDP)
	fmt.Printf("Namespace %s deleted \n", namespaceUDP)
	avgS, _, _, _, _ := utils.AvgSpeed(netSpeeds, cpuClie, cpuServ, cpuconfC, cpuconfS, float64(iteration))
	return fmt.Sprintf("%f", avgS) + " Gbits/sec" /*+
	" client cpu usage: " + fmt.Sprintf("%f", avgCPUC) + " and server CPU usage: " + fmt.Sprintf("%f", avgCPUS)*/
}

func parseVelServiceUdp(str string, clientset *kubernetes.Clientset, udp string) []float64 {
	var velspeed []float64
	velspeed = make([]float64, 3)
	var errConv error
	strs := strings.Split(str, "Client connecting")
	for i := 0; i < 3; i++ {
		vectString := strings.Split(strs[i+1], "0.0-10.")
		substring := strings.Split(vectString[1], "\n")
		substring = strings.Split(substring[0], " ")
		fmt.Println(substring[len(substring)-1])
		fmt.Println(substring[len(substring)-2])
		if strings.Contains(substring[len(substring)-2], " ") {
			substring[len(substring)-2] = strings.Replace(substring[len(substring)-2], " ", "0", 4)
		}
		velspeed[i], errConv = strconv.ParseFloat(substring[len(substring)-2], 64)
		if errConv != nil {
			fmt.Println("errore alla riga 437 di iperfService")
			panic(errConv)
		}
		switch substring[len(substring)-1] {
		case "Mbits/sec":
			velspeed[i] = velspeed[i] / 1000
		case "Kbits/sec":
			velspeed[i] = velspeed[i] / 1000000
		case "Gbits/sec":
			fmt.Println("Ok, Gbits/sec")
		}

	}
	return velspeed
}

func createTCPService(clientset *kubernetes.Clientset, label string) *apiv1.Service {

	svc := apiv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService,
			Namespace: namespace,
			//Labels: map[string]string{"":""},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:       "tcpport",
				Protocol:   "TCP",
				Port:       5001,
				TargetPort: intstr.IntOrString{intstr.Type(0), 5001, "5001"},
			}},
			Selector: map[string]string{"app": label},
		},
	}
	svcCr, errCr := clientset.CoreV1().Services(namespace).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if errCr != nil {
		panic(errCr)
	}
	fmt.Println("Service my-service-iperf created " + svcCr.GetName())
	return svcCr
}
