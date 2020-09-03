package iperf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"os"
	"strconv"
	"strings"
	"testk8s/utils"
	"time"
)

var deplName = "serveriperf3"
var namespace = "testiperf"
var jobName = "jobiperfclient"
var image = "networkstatic/iperf3"
var labelServer = "iperfserver"
var labelClient = "iperfclient"
var iteration = 12
var node = ""
var namePol string
var node2 = "node2"
var networkPolicies *v1.NetworkPolicy

var netSpeeds []float64
var cpuServ []float64
var cpuconfC []float64
var cpuClie []float64
var cpuconfS []float64

func IperfTCPPodtoPod(clientset *kubernetes.Clientset, casus int, fileoutput *os.File, netpol bool, numNetPol int) string {

	node = utils.SetNodeSelector(casus)
	nsSpec := utils.CreateNS(clientset, namespace)
	fmt.Printf("Namespace %s created\n", nsSpec.Name)
	netSpeeds := make([]float64, iteration)
	cpuServ := make([]float64, iteration)
	cpuClie := make([]float64, iteration)
	cpuconfC := make([]float64, iteration)
	cpuconfS := make([]float64, iteration)

	if netpol {
		utils.CreateBulk(0, numNetPol, clientset, namespace)
	}
	//create one deployment of iperf server
	//todo vedere come poter velocizzare con nomi diversi per deployment etc (tipo random string per ogni deployment
	part := 0
	for i := 0; i <= (iteration / 4); i++ {
		commandD := "iperf3 -s -p 5002 -V"
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
		podI, errPodSearch := clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if errors.IsNotFound(errPodSearch) {
			fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
		} else if statusError, isStatus := errPodSearch.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %s in namespace %s: %v\n",
				pod, namespace, statusError.ErrStatus.Message)
		} else if errPodSearch != nil {
			panic(errPodSearch.Error())
		} else {
			fmt.Printf("Found pod %s in namespace %s\n", pod.Name, namespace)
			podIP := podI.Status.PodIP
			fmt.Printf("Server IP: %s\n", podIP)

		}

		if netpol {
			namePol = utils.CreateAllNetPol(clientset, numNetPol, namespace, labelServer, labelClient)
		}

		command := "for i in 0 1 2; do iperf3 -c " + podI.Status.PodIP + " -p 5002 -V -N -t 10 -Z -A 1,2 -M 1448 >> file.txt; sleep 11; done; cat file.txt"
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
								Image:   "networkstatic/iperf3",
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

		fmt.Println(str)
		if strings.Contains(str, "error - unable") {
			i = i - 1
			utils.CleanCluster(clientset, namespace, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
			continue
		}
		//works on strings

		velspeed, cpuClient, cpuServer := parseVel(str, clientset, namespace)

		fmt.Printf("%d %f Gbits/sec 	clientcpu: %f	server cpu :%f\n ", i, velspeed, cpuClient, cpuServer)
		//todo vedere cosa succede con float 32, per ora 64
		for j := 0; j < 3; j++ {
			netSpeeds[part] = velspeed[j]
			cpuClie[part] = cpuClient[j]
			cpuServ[part] = cpuServer[j]
			part++
		}
		//Clean the cluster

		utils.CleanCluster(clientset, namespace, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
		if netpol {
			utils.DeleteAllPolicies(clientset, namespace, namePol)
		}
	}

	if netpol {
		utils.DeleteBulk(0, numNetPol, clientset, namespace)
	}

	utils.DeleteNS(clientset, namespace)
	fmt.Printf("Test Namespace: %s deleted\n", namespace)
	time.Sleep(10 * time.Second)
	avgS, avgCPUC, avgCPUS, _, _ := utils.AvgSpeed(netSpeeds, cpuClie, cpuServ, cpuconfC, cpuconfS, float64(iteration))
	return fmt.Sprintf("%f", avgS) + " Gbits/sec, client cpu usage: " + fmt.Sprintf("%f", avgCPUC) + " and server CPU usage: " + fmt.Sprintf("%f", avgCPUS)

}

func IperfUDPPodtoPod(clientset *kubernetes.Clientset, casus int, fileoutput *os.File, netpol bool, numNetPol int) string {

	node = utils.SetNodeSelector(casus)
	utils.CreateNS(clientset, namespaceUDP)

	fmt.Println("Namespace UDP testiperf created")

	netSpeeds := make([]float64, iteration)
	cpuServ := make([]float64, iteration)
	cpuClie := make([]float64, iteration)
	cpuconfC := make([]float64, iteration)
	cpuconfS := make([]float64, iteration)

	//create one deployment of iperf server UDP
	part := 0
	for i := 0; i <= (iteration / 4); i++ {
		commandD := "iperf3 -s -p 5003 -V"
		dep := createIperfDeployment(namespaceUDP, image, commandD)
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
		podI, errPodSearch := clientset.CoreV1().Pods(namespaceUDP).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if errors.IsNotFound(errPodSearch) {
			fmt.Printf("Pod %s in namespace %s not found\n", pod, namespaceUDP)
		} else if statusError, isStatus := errPodSearch.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %s in namespace %s: %v\n",
				pod, namespaceUDP, statusError.ErrStatus.Message)
		} else if errPodSearch != nil {
			panic(errPodSearch.Error())
		} else {
			fmt.Printf("Found pod %s in namespace %s\n", pod.Name, namespaceUDP)
			podIP := podI.Status.PodIP
			fmt.Printf("UDP Server IP: %s\n", podIP)

		}

		command := "for i in 0 1 2; do iperf3 -c " + podI.Status.PodIP + " -u -b 0 -p 5003 -V -N -t 10 -Z -M 1448 >> file.txt; sleep 11; done; cat file.txt"
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
					fileoutput.WriteString(str)
					fmt.Println(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		if strings.Contains(str, "error - unable") {
			i = i - 1
			utils.CleanCluster(clientset, namespaceUDP, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
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
		utils.CleanCluster(clientset, namespaceUDP, "app=iperfserver", "app=iperfclient", deplName, jobName, pod.Name)
	}

	utils.DeleteNS(clientset, namespaceUDP)
	time.Sleep(10 * time.Second)
	avgS, avgCPUC, avgCPUS, _, _ := utils.AvgSpeed(netSpeeds, cpuClie, cpuServ, cpuconfC, cpuconfS, float64(iteration))
	return fmt.Sprintf("%f", avgS) + " Gbits/sec, client cpu usage: " + fmt.Sprintf("%f", avgCPUC) + " and server CPU usage: " + fmt.Sprintf("%f", avgCPUS)
}

func parseVel(strs string, clientset *kubernetes.Clientset, ns string) ([]float64, []float64, []float64) {
	var velspeed, clientCPU, serverCPU []float64
	velspeed = make([]float64, 3)
	clientCPU = make([]float64, 3)
	serverCPU = make([]float64, 3)
	var errConv error
	str := strings.Split(strs, "iperf Done.\n")
	for i := 0; i < 3; i++ {
		if strings.Contains(strs, "Connection timed out") || strings.Contains(str[i+1], "Connection refused") || strings.Contains(str[i+1], "Connection timed out") {
			utils.DeleteNS(clientset, ns)
			panic("error in client server communication")
		} else {
			vectString := strings.Split(str[i+1], "0.00-10.00 ")
			substringSpeed := strings.Split(vectString[1], "/sec")
			vectString[len(vectString)-1] = strings.Replace(vectString[len(vectString)-1], "%", "0", 5)
			substringCPU := strings.Split(vectString[len(vectString)-1], "(")
			speedPos := strings.Split(substringSpeed[0], " ")
			speed := speedPos[len(speedPos)-2]
			if strings.Contains(speed, " ") {
				strings.Replace(speed, " ", "0", 2)
			}
			cpuSend := strings.Split(substringCPU[len(substringCPU)-3], " ")
			cpuServ := strings.Split(substringCPU[len(substringCPU)-2], " ")
			clientCPU[i], errConv = strconv.ParseFloat(cpuSend[len(cpuSend)-2], 64)
			if errConv != nil {
				fmt.Println("Errore nel client conversion cpu + " + cpuSend[len(cpuSend)-2])
				panic(errConv)
			}
			serverCPU[i], errConv = strconv.ParseFloat(cpuServ[len(cpuServ)-2], 64)
			if errConv != nil {
				fmt.Println("Errore nel server conversion cpu " + cpuServ[len(cpuServ)-2])
				panic(errConv)
			}
			velspeed[i], errConv = strconv.ParseFloat(speed, 64)
			if errConv != nil {
				fmt.Println("Errore nel speed conversion " + speed)
				panic(errConv)
			}

			switch speedPos[len(speedPos)-1] {
			case "Mbits":
				velspeed[i] = velspeed[i] / 1000
			case "Kbits":
				velspeed[i] = velspeed[i] / 1000000
			case "Gbits":
				fmt.Println("Ok, Gbits/sec")
			}
		}
	}
	fmt.Println(velspeed)
	return velspeed, clientCPU, serverCPU
}

func createIperfDeployment(ns string, imageDepl string, command string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "iperfserver"},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "iperfserver",
					Labels: map[string]string{"app": "iperfserver"},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "iperf3server",
							Image:   imageDepl,
							Command: []string{"/bin/sh"},
							Args:    []string{"-c", command},
						},
					},
					NodeSelector: map[string]string{"type": node2},
				},
			},
		},
	}
}
