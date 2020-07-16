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
	"os"
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

/*var netSpeeds []float64
var confidenceArray []float64
var confidenceArrayCpuC []float64
var confidenceArrayCpuS []float64
var cpuC []float64
var cpuS []float64
*/

var netSpeeds float64
var confidenceArray float64
var confidenceArrayCpuC float64
var confidenceArrayCpuS float64
var cpuC float64
var cpuS float64

func NetperfTCPPodtoPod(clientset *kubernetes.Clientset, casus int, fileoutput *os.File) string {

	node = utils.SetNodeSelector(casus)
	nsCR := utils.CreateNS(clientset, namespace)
	fmt.Printf("Namespace %s created\n", nsCR.Name)
	best := "10000.0"
	//create one deployment of netperf server
	/*
		netSpeeds := make([]float64, iteration)
		confidenceArray := make([]float64, iteration)
		confidenceArrayCpuC := make([]float64, iteration)
		confidenceArrayCpuS := make([]float64, iteration)
		cpuC := make([]float64, iteration)
		cpuS := make([]float64, iteration)
	*/
	for i := 0; i < 5; i++ {

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

		} //2 3 4
		command := "netperf -H " + podI.Status.PodIP + " -T 1,2 -i 30,2 -j -p 15001 -c -C -- -D > file.txt; cat file.txt "
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
					//TODO da mettere su file invece che stampare
					fmt.Println(str)
					fileoutput.WriteString(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}
		if i <= 4 && strings.Contains(str, "!!! WARNING") {
			best = bestMeasure(str, best)
			utils.CleanCluster(clientset, namespace, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)
			if i < 4 {
				continue
			}
		} else {
			best = str
		}

		fmt.Printf("misura trovata all'iterazione: %d\n", i)
		i = 5
		//netSpeeds, confidenceArray, cpuC, cpuS, confidenceArrayCpuC, confidenceArrayCpuS = calculateSpeed(str, clientset, namespace, 0)

		//todo vedere cosa succede con float 32, per ora 64

		utils.CleanCluster(clientset, namespace, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)
	}
	utils.DeleteNS(clientset, namespace)
	netSpeeds, confidenceArray, cpuC, cpuS, confidenceArrayCpuC, confidenceArrayCpuS = calculateSpeed(best, clientset, namespace, 0)
	//avgSp, avgClient, avgServer, CpuPercCl, CpuPercS := utils.AvgSpeed(netSpeeds, cpuC, cpuS, confidenceArrayCpuC, confidenceArrayCpuS, float64(iteration))
	//return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + "error: " + fmt.Sprintf("%f", CpuPercCl) + "/" + fmt.Sprintf("%f", avgServer) + ":" + fmt.Sprintf("%f", CpuPercS)
	return fmt.Sprintf("%f", netSpeeds) + " Gbits/sec, confidence avg: " + fmt.Sprintf("%f", confidenceArray) + " and cpu client usage : " + fmt.Sprintf("%f", cpuC) + "error: " + fmt.Sprintf("%f", confidenceArrayCpuC) + "/server: " + fmt.Sprintf("%f", cpuS) + ":" + fmt.Sprintf("%f", confidenceArrayCpuS)
}

func NetperfUDPPodtoPod(clientset *kubernetes.Clientset, casus int, fileoutput *os.File) string {

	node = utils.SetNodeSelector(casus)
	best := "10000.0"
	nsCR := utils.CreateNS(clientset, namespaceUDP)
	fmt.Printf("Namespace UDP %s created\n", nsCR.Name)
	/*
		netSpeeds := make([]float64, iteration)
		confidenceArray := make([]float64, iteration)
		confidenceArrayCpuC := make([]float64, iteration)
		confidenceArrayCpuS := make([]float64, iteration)
		cpuC := make([]float64, iteration)
		cpuS := make([]float64, iteration)*/
	//create one deployment of netperf server UDP
	for i := 0; i < 5; i++ {
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

		command := "netperf -t UDP_STREAM -H " + podI.Status.PodIP + " -i 30,2 -p 15003 -v 2 -c -C -T 1,2 -- -R 1 -D > file.txt; cat file.txt"
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
					fileoutput.WriteString(str)
					//TODO stampare su file
					fmt.Println(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		if i <= 4 && strings.Contains(str, "!!! WARNING") {
			best = bestMeasure(str, best)
			utils.CleanCluster(clientset, namespaceUDP, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)
			if i < 4 {
				continue
			}
		} else {
			best = str
		}
		fmt.Printf("misura trovata all'iterazione: %d\n", i)
		i = 5
		//works on strings
		//todo vedere cosa succede con float 32, per ora 64
		utils.CleanCluster(clientset, namespaceUDP, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)
	}
	utils.DeleteNS(clientset, namespaceUDP)
	/*avgSp, avgClient, avgServer, CpuPercCl, CpuPercS := utils.AvgSpeed(netSpeeds, cpuC, cpuS, confidenceArrayCpuC, confidenceArrayCpuS, float64(iteration))
	return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + ":" + fmt.Sprintf("%f", CpuPercCl) + " / " + fmt.Sprintf("%f", avgServer) + ":" + fmt.Sprintf("%f", CpuPercS)
	*/
	netSpeeds, confidenceArray, cpuC, cpuS, confidenceArrayCpuC, confidenceArrayCpuS = calculateSpeed(best, clientset, namespaceUDP, -1)
	return fmt.Sprintf("%f", netSpeeds) + " Gbits/sec, confidence avg: " + fmt.Sprintf("%f", confidenceArray) + " and cpu client usage : " + fmt.Sprintf("%f", cpuC) + "error: " + fmt.Sprintf("%f", confidenceArrayCpuC) + "/server: " + fmt.Sprintf("%f", cpuS) + ":" + fmt.Sprintf("%f", confidenceArrayCpuS)
}

func bestMeasure(str string, best string) string {
	var minTotal float64
	var minMeasure float64
	var errConv error

	if best == "10000.0" {
		minTotal, errConv = strconv.ParseFloat(best, 64)
		if errConv != nil {
			fmt.Println("errore di conversione linea 427")
			panic(errConv)
		}
	} else {
		s := strings.Split(best, "Throughput")
		selected := strings.Split(s[1], "%")[0]
		throughput := strings.Replace(selected, " ", "0", 10)
		throughput = strings.Replace(throughput, ":", "0", 1)
		minTotal, errConv = strconv.ParseFloat(throughput, 64)
		if errConv != nil {
			fmt.Println("errore di conversione linea 440")
			panic(errConv)
		}
	}

	s := strings.Split(str, "Throughput")
	selected := strings.Split(s[1], "%")[0]
	throughput := strings.Replace(selected, " ", "0", 10)
	throughput = strings.Replace(throughput, ":", "0", 1)
	minMeasure, errConv = strconv.ParseFloat(throughput, 64)
	if errConv != nil {
		fmt.Println("errore di conversione linea 440")
		panic(errConv)
	}
	if minMeasure < minTotal {
		best = str
	}

	return best
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

/*
func calculateSpeed(str string, clientset *kubernetes.Clientset, ns string, add int) ([]float64, []float64, []float64, []float64, []float64, []float64) {
	var velspeed []float64
	var conf []float64
	var cpuC []float64
	var confCpuC []float64
	var confCpuS []float64
	var cpuS []float64
	var errConv error

	velspeed = make([]float64, iteration)
	conf = make([]float64, iteration)
	cpuC = make([]float64, iteration)
	cpuS = make([]float64, iteration)
	confCpuC = make([]float64, iteration)
	confCpuS = make([]float64, iteration)

	//works on strings
	iterationString := strings.Split(str, "MIGRATED")
	for i := 1; i <= iteration; i++ {

		if strings.Contains(iterationString[i], "Connection refused") || strings.Contains(iterationString[i], "establish control") || strings.Contains(iterationString[i], "Connection time out") {
			utils.DeleteNS(clientset, ns)
			panic("establish control: are you sure there is a netserver listening on 10.103.45.178 at port 15001?")
		} else {
			if strings.Contains(iterationString[i], "!!! WARNING") && add == -1 {
				vectString := strings.Split(iterationString[i], "\n")
				conf[i-1], confCpuC[i-1], confCpuS[i-1] = warnings(vectString)
				strspeed := strings.Split(vectString[14+add], "    ")
				if strings.Contains(strspeed[3], " ") {
					strspeed[3] = strings.Replace(strspeed[3], " ", "0", 5)
				}
				velspeed[i-1], errConv = strconv.ParseFloat(strspeed[3], 64)
				if errConv != nil {
					fmt.Println("ERRORE Warning: " + strspeed[3])
					panic(errConv)
				}
				if strings.Contains(strspeed[4], " ") {
					strspeed[4] = strings.Replace(strspeed[4], " ", "0", 5)
				}
				cpuC[i-1], errConv = strconv.ParseFloat(strspeed[3], 64)
				if errConv != nil {
					fmt.Println("ERRORE Warning: " + strspeed[3])
					panic(errConv)
				}
				strspeed = strings.Split(vectString[15+add], "    ")
				if strings.Contains(strspeed[len(strspeed)-2], " ") {
					strspeed[len(strspeed)-2] = strings.Replace(strspeed[len(strspeed)-2], " ", "0", 5)
				}
				cpuS[i-1], errConv = strconv.ParseFloat(strspeed[len(strspeed)-2], 64)
				if errConv != nil {
					fmt.Println("ERRORE Warning: " + strspeed[len(strspeed)-2])
					panic(errConv)
				}
			} else {
				if strings.Contains(iterationString[i], "!!! WARNING") {
					vectString := strings.Split(iterationString[i], "\n")
					conf[i-1], confCpuC[i-1], confCpuS[i-1] = warnings(vectString)
					strspeed := strings.Split(vectString[14], "    ")
					strspeeds := strings.Split(strspeed[2], "  ")
					if strings.Contains(strspeeds[1], " ") {
						strspeeds[1] = strings.Replace(strspeeds[1], " ", "0", 5)
					}
					velspeed[i-1], errConv = strconv.ParseFloat(strspeeds[1], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning: " + strspeeds[1])
						panic(errConv)
					}
					if strings.Contains(strspeeds[2], " ") {
						strspeeds[2] = strings.Replace(strspeeds[2], " ", "0", 5)
					}
					cpuC[i-1], errConv = strconv.ParseFloat(strspeeds[2], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning: " + strspeeds[2])
						panic(errConv)
					}
					fmt.Printf("arrivo qui %d\n", len(strspeed[3]))
					if strings.Contains(strspeed[3], " ") {
						strspeed[3] = strings.Replace(strspeed[3], " ", "0", 5)
					}
					cpuS[i-1], errConv = strconv.ParseFloat(strspeed[3], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning: " + strspeed[3])
						panic(errConv)
					}
					fmt.Printf("supero senza motivi apparenti \n")

				} else {
					vectString := strings.Split(iterationString[i], "\n")
					if add == -1 {
						strspeed := strings.Split(vectString[6+add], "   ")
						if strings.Contains(strspeed[7], " ") {
							strspeed[7] = strings.Replace(strspeed[7], " ", "0", 3)
						}
						velspeed[i-1], errConv = strconv.ParseFloat(strspeed[7], 64)
						if errConv != nil {
							fmt.Println("ERRORE Warning: " + strspeed[7])
							panic(errConv)
						}
						if strings.Contains(strspeed[8], " ") {
							strspeed[8] = strings.Replace(strspeed[8], " ", "0", 3)
						}
						cpuC[i-1], errConv = strconv.ParseFloat(strspeed[8], 64)
						if errConv != nil {
							fmt.Println("ERRORE Warning: " + strspeed[8])
							panic(errConv)
						}
						strspeed = strings.Split(vectString[7+add], "   ")
						if strings.Contains(strspeed[len(strspeed)-2], " ") {
							strspeed[len(strspeed)-2] = strings.Replace(strspeed[len(strspeed)-2], " ", "0", 3)
						}
						cpuS[i-1], errConv = strconv.ParseFloat(strspeed[len(strspeed)-2], 64)
						if errConv != nil {
							fmt.Println("ERRORE Warning line 565: " + strspeed[len(strspeed)-2])
							panic(errConv)
						}
					} else {
						strspeed := strings.Split(vectString[6], "  ")
						fmt.Println(" ecco qui " + strspeed[7])
						fmt.Println(" ecco qui " + strspeed[8])
						fmt.Println(" ecco qui " + strspeed[10])
						if strings.Contains(strspeed[7], " ") {
							strspeed[7] = strings.Replace(strspeed[7], " ", "0", 5)
						}
						velspeed[i-1], errConv = strconv.ParseFloat(strspeed[7], 64)
						if errConv != nil {
							fmt.Println("ERRORE Warning: " + strspeed[7])
							panic(errConv)
						}
						if strings.Contains(strspeed[8], " ") {
							strspeed[8] = strings.Replace(strspeed[8], " ", "0", 5)
						}
						cpuC[i-1], errConv = strconv.ParseFloat(strspeed[8], 64)
						if errConv != nil {
							fmt.Println("ERRORE Warning: " + strspeed[8])
							panic(errConv)
						}
						if strings.Contains(strspeed[10], " ") {
							strspeed[10] = strings.Replace(strspeed[10], " ", "0", 5)
						}
						cpuS[i-1], errConv = strconv.ParseFloat(strspeed[10], 64)
						if errConv != nil {
							fmt.Println("ERRORE Warning: " + strspeed[10])
							panic(errConv)
						}
					}
					conf[i-1] = 2.5
					confCpuC[i-1] = 2.5
					confCpuS[i-1] = 2.5
					if errConv != nil {
						fmt.Println("ERRORE: in conversione da stringa a float prendi valore sbagliato")
						panic(errConv)
					}
				}

			}
			speed := "Mbits/sec"
			switch speed {
			case "Mbits/sec":
				velspeed[i-1] = velspeed[i-1] / 1000
			case "Kbits/sec":
				velspeed[i-1] = velspeed[i-1] / 1000000
			case "Gbits/sec":
				fmt.Println("Ok, Gbits/sec")

			}
		}
	}

	return velspeed, conf, cpuC, cpuS, confCpuC, confCpuS
}
*/

func calculateSpeed(str string, clientset *kubernetes.Clientset, ns string, add int) (float64, float64, float64, float64, float64, float64) {
	var velspeed float64
	var conf float64
	var cpuC float64
	var confCpuC float64
	var confCpuS float64
	var cpuS float64
	var errConv error

	//works on strings

	if strings.Contains(str, "Connection refused") || strings.Contains(str, "establish control") || strings.Contains(str, "Connection time out") {
		utils.DeleteNS(clientset, ns)
		panic("establish control: are you sure there is a netserver listening on 10.103.45.178 at port 15001?")
	} else {
		if strings.Contains(str, "!!! WARNING") && add == -1 {
			vectString := strings.Split(str, "\n")
			conf, confCpuC, confCpuS = warnings(vectString, 0)
			strspeed := strings.Split(vectString[14+add], "    ")
			if strings.Contains(strspeed[3], " ") {
				strspeed[3] = strings.Replace(strspeed[3], " ", "0", 5)
			}
			velspeed, errConv = strconv.ParseFloat(strspeed[3], 64)
			if errConv != nil {
				fmt.Println("ERRORE Warning: " + strspeed[3])
				panic(errConv)
			}
			if strings.Contains(strspeed[4], " ") {
				strspeed[4] = strings.Replace(strspeed[4], " ", "0", 5)
			}
			cpuC, errConv = strconv.ParseFloat(strspeed[4], 64)
			if errConv != nil {
				fmt.Println("ERRORE Warning: " + strspeed[4])
				panic(errConv)
			}
			strspeed = strings.Split(vectString[15+add], "    ")
			if strings.Contains(strspeed[len(strspeed)-2], " ") {
				strspeed[len(strspeed)-2] = strings.Replace(strspeed[len(strspeed)-2], " ", "0", 5)
			}
			cpuS, errConv = strconv.ParseFloat(strspeed[len(strspeed)-2], 64)
			if errConv != nil {
				fmt.Println("ERRORE Warning: " + strspeed[len(strspeed)-2])
				panic(errConv)
			}
		} else {
			if strings.Contains(str, "!!! WARNING") {
				vectString := strings.Split(str, "\n")
				conf, confCpuC, confCpuS = warnings(vectString, add)
				strspeed := strings.Split(vectString[14+add], "    ")
				strspeeds := strings.Split(strspeed[2], "  ")
				if strings.Contains(strspeeds[1], " ") {
					strspeeds[1] = strings.Replace(strspeeds[1], " ", "0", 5)
				}
				velspeed, errConv = strconv.ParseFloat(strspeeds[1], 64)
				if errConv != nil {
					fmt.Println("ERRORE Warning: " + strspeeds[1])
					panic(errConv)
				}
				if strings.Contains(strspeeds[2], " ") {
					strspeeds[2] = strings.Replace(strspeeds[2], " ", "0", 5)
				}
				cpuC, errConv = strconv.ParseFloat(strspeeds[2], 64)
				if errConv != nil {
					fmt.Println("ERRORE Warning: " + strspeeds[2])
					panic(errConv)
				}
				fmt.Printf("arrivo qui %d\n", len(strspeed[3]))
				if strings.Contains(strspeed[3], " ") {
					strspeed[3] = strings.Replace(strspeed[3], " ", "0", 5)
				}
				cpuS, errConv = strconv.ParseFloat(strspeed[3], 64)
				if errConv != nil {
					fmt.Println("ERRORE Warning: " + strspeed[3])
					panic(errConv)
				}
				fmt.Printf("supero senza motivi apparenti \n")

			} else {
				vectString := strings.Split(str, "\n")
				if add == -1 {

					strspeed := strings.Split(vectString[5], "   ")
					if strings.Contains(strspeed[7], " ") {
						strspeed[7] = strings.Replace(strspeed[7], " ", "0", 3)
					}
					velspeed, errConv = strconv.ParseFloat(strspeed[7], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning linea 737: " + strspeed[7])
						panic(errConv)
					}
					if strings.Contains(strspeed[8], " ") {
						strspeed[8] = strings.Replace(strspeed[8], " ", "0", 3)
					}
					cpuC, errConv = strconv.ParseFloat(strspeed[8], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning 745: " + strspeed[8])
						panic(errConv)
					}
					strspeed = strings.Split(vectString[7+add], "   ")
					if strings.Contains(strspeed[len(strspeed)-2], " ") {
						strspeed[len(strspeed)-2] = strings.Replace(strspeed[len(strspeed)-2], " ", "0", 3)
					}
					cpuS, errConv = strconv.ParseFloat(strspeed[len(strspeed)-2], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning line 565: " + strspeed[len(strspeed)-2])
						panic(errConv)
					}
				} else {
					strspeed := strings.Split(vectString[6+add], "  ")
					fmt.Println(" ecco qui " + strspeed[7])
					fmt.Println(" ecco qui " + strspeed[8])
					fmt.Println(" ecco qui " + strspeed[10])
					if strings.Contains(strspeed[7], " ") {
						strspeed[7] = strings.Replace(strspeed[7], " ", "0", 5)
					}
					velspeed, errConv = strconv.ParseFloat(strspeed[7], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning: " + strspeed[7])
						panic(errConv)
					}
					if strings.Contains(strspeed[8], " ") {
						strspeed[8] = strings.Replace(strspeed[8], " ", "0", 5)
					}
					cpuC, errConv = strconv.ParseFloat(strspeed[8], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning: " + strspeed[8])
						panic(errConv)
					}
					if strings.Contains(strspeed[10], " ") {
						strspeed[10] = strings.Replace(strspeed[10], " ", "0", 5)
					}
					cpuS, errConv = strconv.ParseFloat(strspeed[10], 64)
					if errConv != nil {
						fmt.Println("ERRORE Warning: " + strspeed[10])
						panic(errConv)
					}
				}
				conf = 2.5
				confCpuC = 2.5
				confCpuS = 2.5
				if errConv != nil {
					fmt.Println("ERRORE: in conversione da stringa a float prendi valore sbagliato")
					panic(errConv)
				}
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

	return velspeed, conf, cpuC, cpuS, confCpuC, confCpuS
}

func warnings(vectString []string, add int) (float64, float64, float64) {
	throughput := strings.Split(vectString[5+add], ":")
	throughput[len(throughput)-1] = strings.Replace(throughput[len(throughput)-1], "%", "0", 1)
	throughput[len(throughput)-1] = strings.Replace(throughput[len(throughput)-1], " ", "0", 3)
	fmt.Printf("%s\n", throughput[len(throughput)-1])
	ret1, err1 := strconv.ParseFloat(throughput[len(throughput)-1], 64)
	if err1 != nil {
		fmt.Printf("ERRORE Warning: %f\n ", ret1)
		panic(ret1)
	}
	errCpuC := strings.Split(vectString[6+add], ":")
	errCpuC[len(errCpuC)-1] = strings.Replace(errCpuC[len(errCpuC)-1], "%", "0", 1)
	errCpuC[len(errCpuC)-1] = strings.Replace(errCpuC[len(errCpuC)-1], " ", "0", 3)
	ret2, err2 := strconv.ParseFloat(errCpuC[len(errCpuC)-1], 64)
	if err2 != nil {
		fmt.Printf("ERRORE Warning: %f\n ", ret2)
		panic(ret2)
	}
	errCpuS := strings.Split(vectString[7+add], ":")
	errCpuS[len(errCpuS)-1] = strings.Replace(errCpuS[len(errCpuS)-1], "%", "0", 1)
	errCpuS[len(errCpuS)-1] = strings.Replace(errCpuS[len(errCpuS)-1], " ", "0", 3)
	ret3, err3 := strconv.ParseFloat(errCpuS[len(errCpuS)-1], 64)
	if err3 != nil {
		fmt.Printf("ERRORE Warning: %f\n ", ret3)
		panic(ret3)
	}
	fmt.Printf("sto per ritornare! %f %f %f\n ", ret1, ret2, ret3)
	return ret1, ret2, ret3
}
