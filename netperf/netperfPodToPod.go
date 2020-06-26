package netperf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"strconv"
	"strings"
	utils "testk8s/utils"
)

var deplName = "servernetperf"
var nameService = "my-service-netperf"
var jobName = "jobnetperfclient"
var namespace = "testnetperftcp"
var namespaceUDP = "testnetperfudp"
var iteration = 5
var node = " "
var node2 = "node2"

var netSpeeds []float64
var confidenceArray []float64
var cpuC []float64
var cpuS []float64

func NetperfTCPPodtoPod(clientset *kubernetes.Clientset, casus int) string {

	node = utils.SetNodeSelector(casus)
	nsCR := utils.CreateNS(clientset, namespace)
	fmt.Printf("Namespace %s created\n", nsCR.Name)

	//create one deployment of netperf server

	netSpeeds := make([]float64, iteration)
	confidenceArray := make([]float64, iteration)
	cpuC := make([]float64, iteration)
	cpuS := make([]float64, iteration)

	for i := 0; i < iteration; i++ {
		dep := createNetperfServer("15001", namespace)
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

		podvect, errP := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if errP != nil {
			panic(errP)
		}
		fmt.Print("Wait for pod creation..")
		for {
			podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if errP != nil {
				panic(errP)
			}
			var num = len(podvect.Items)
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
					podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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
				pod.Name, namespace, statusError.ErrStatus.Message)
		} else if errPodSearch != nil {
			panic(errPodSearch.Error())
		} else {
			fmt.Printf("Found pod %s in namespace %s\n", pod.Name, namespace)
			podIP := podI.Status.PodIP
			fmt.Printf("Server IP: %s\n", podIP)

		}
		command := "netperf -H " + podI.Status.PodIP + " -i 30,2 -j -p 15001 -v 2 -c -- -D > file.txt; cat file.txt"
		fmt.Println("Creating Netperf Client: " + command)
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
						Name:   "netperfclient",
						Labels: map[string]string{"app": "netperfclient"},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:    "netperfclient",
								Image:   "leannet/k8s-netperf",
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
		podClient, errC := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
		if errC != nil {
			panic(errC)
		}
		for {
			if len(podClient.Items) != 0 {
				break
			}
			podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
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
					podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
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
					//TODO da cancellare
					fmt.Println(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		velspeed, conf := calculateSpeed(str, clientset, namespace, 0)
		fmt.Printf("%d %f Gbits/sec \n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		netSpeeds[i] = velspeed
		confidenceArray[i] = conf
		cpuC[i] = float64(i)
		cpuS[i] = float64(i)

		utils.CleanCluster(clientset, namespace, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)
	}

	utils.DeleteNS(clientset, namespace)
	avgSp, avgClient, avgServer := utils.AvgSpeed(netSpeeds, cpuC, cpuS, float64(iteration))
	return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + "/" + fmt.Sprintf("%f", avgServer)
}

func NetperfUDPPodtoPod(clientset *kubernetes.Clientset, casus int) string {

	node = utils.SetNodeSelector(casus)
	nsCR := utils.CreateNS(clientset, namespaceUDP)
	fmt.Printf("Namespace UDP %s created\n", nsCR.Name)

	netSpeeds := make([]float64, iteration)
	confidenceArray := make([]float64, iteration)
	//create one deployment of netperf server UDP

	for i := 0; i < iteration; i++ {
		dep := createNetperfServer("15003", namespaceUDP)
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

		command := "netperf -t UDP_STREAM -H " + podI.Status.PodIP + " -i 30,2 -p 15003 -v 2 -c -- -R 1 -D > file.txt; cat file.txt"
		fmt.Println("Creating UDP Netperf Client: " + command)
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
						Name:   "netperfclient",
						Labels: map[string]string{"app": "netperfclient"},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:    "netperfclient",
								Image:   "leannet/k8s-netperf",
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
		podClient, errC := clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
		if errC != nil {
			panic(errC)
		}
		for {
			if len(podClient.Items) != 0 {
				break
			}
			podClient, errC = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
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
					podClient, errC = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
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
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		velspeed, conf := calculateSpeed(str, clientset, namespaceUDP, 0)
		fmt.Printf("%d %f Gbits/sec \n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		netSpeeds[i] = velspeed
		confidenceArray[i] = conf
		cpuC[i] = float64(i)
		cpuS[i] = float64(i)

		utils.CleanCluster(clientset, namespaceUDP, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)
	}

	utils.DeleteNS(clientset, namespaceUDP)
	avgSp, avgClient, avgServer := utils.AvgSpeed(netSpeeds, cpuC, cpuS, float64(iteration))
	return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + "/" + fmt.Sprintf("%f", avgServer)
}

func confidenceAVG(speeds []float64, conf []float64, div float64) float64 {
	var max = 0.00
	var min = 10000.00
	var j, k = 0, 0

	for i := 0; i < int(div); i++ {
		if speeds[i] > max {
			max = speeds[i]
			k = i
		}
		if speeds[i] < min {
			j = i
			min = speeds[i]
		}
	}

	conf[j] = 0.0
	conf[k] = 0.0
	var sum = 0.0

	for i := 0; i < int(div); i++ {
		sum = sum + conf[i]
	}
	div = div - 2.0

	return sum / div
}

func createNetperfServer(port string, ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "netperfserver"},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "netperfserver",
					Labels: map[string]string{"app": "netperfserver"},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "netperfserver",
							Image:   "leannet/k8s-netperf",
							Command: []string{"/bin/sh"},
							Args:    []string{"-c", "netserver -p " + port + " -v 2 -d; tail -f /dev/null"},
						},
					},
					NodeSelector: map[string]string{"type": node2},
				},
			},
		},
	}
}

func calculateSpeed(str string, clientset *kubernetes.Clientset, ns string, add int) (float64, float64) {
	var velspeed float64
	var conf float64
	var errConv error
	//works on strings
	if strings.Contains(str, "Connection refused") || strings.Contains(str, "establish control") || strings.Contains(str, "Connection time out") {
		utils.DeleteNS(clientset, ns)
		panic("establish control: are you sure there is a netserver listening on 10.103.45.178 at port 15001?")
	} else {
		if strings.Contains(str, "!!! WARNING") {
			vectString := strings.Split(str, "\n")
			strspeed := strings.Split(vectString[14+add], "    ")
			strspeed = strings.Split(strspeed[2], " ")
			velspeed, errConv = strconv.ParseFloat(strspeed[0], 32)
			if errConv != nil {
				fmt.Println("ERRORE Warning: " + strspeed[0])
				panic(errConv)
			}
		} else {
			vectString := strings.Split(str, "\n")
			if add == -1 {
				strspeed := strings.Split(vectString[6+add], " ")
				velspeed, errConv = strconv.ParseFloat(strspeed[len(strspeed)-1], 32)
			} else {
				strspeed := strings.Split(vectString[6], "  ")
				velspeed, errConv = strconv.ParseFloat(strspeed[len(strspeed)-2], 32)
			}
			conf = 2.5
			if errConv != nil {
				fmt.Println("ERRORE: in conversione da stringa a float prendi valore sbagliato")
				panic(errConv)
			}
		}
		speed := "Mbits/sec"
		switch speed {
		case "Mbits/sec":
			velspeed = velspeed / 1000
		case "Kbits/sec":
			velspeed = velspeed / 1000000
		case "Gbits/sec":
			fmt.Println("Ok, Gbits/sec")

		}
	}

	return velspeed, conf
}
