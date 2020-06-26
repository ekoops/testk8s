package netperf

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
	"testk8s/utils"
)

func TCPservice(clientset *kubernetes.Clientset, casus int, multiple bool) string {

	node = utils.SetNodeSelector(casus)
	svcCr := initializeTCPService(multiple, clientset, namespace, "netperfserver")

	netSpeeds := make([]float64, iteration)
	confidenceArray := make([]float64, iteration)
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

		podvect, errP := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfserver"})
		if errP != nil {
			panic(errP)
		}
		fmt.Print("Wait for pod creation..")
		for {
			podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfserver"})
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
					podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfserver"})
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

		command := "netperf -H " + serviceC.Spec.ClusterIP + " -i 30,2 -j -p 15001 -v 2 -c -- -D -P ,35001> file.txt; cat file.txt"
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
								Name:    "netperfserver",
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
					fmt.Println(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		velspeed, conf := calculateSpeed(str, clientset, namespace, 0)
		fmt.Printf("%d %f Gbits/sec \n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		netSpeeds[i] = velspeed
		confidenceArray[i] = conf
		cpuC[i] = float64(i)
		cpuS[i] = float64(i)

		utils.CleanCluster(clientset, namespace, "app=netperfserver", "app=netperfclient", deplName, jobName, pod.Name)

	}

	utils.DeleteBulk(10, clientset, namespace)
	utils.DeleteNS(clientset, namespace)
	fmt.Printf("Test Namespace: %s deleted\n", namespace)
	avgSp, avgClient, avgServer := utils.AvgSpeed(netSpeeds, cpuC, cpuS, float64(iteration))
	return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + "/" + fmt.Sprintf("%f", avgServer)
}

func UDPservice(clientset *kubernetes.Clientset, casus int, multiple bool) string {

	node = utils.SetNodeSelector(casus)
	nsCr := utils.CreateNS(clientset, namespaceUDP)

	fmt.Printf("Namespace %s created\n", nsCr.Name)

	svc := apiv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService,
			Namespace: namespaceUDP,
			//Labels: map[string]string{"":""},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:       "udpconnectionport",
				Protocol:   "UDP",
				Port:       15201,
				TargetPort: intstr.IntOrString{intstr.Type(0), 15201, "15201"},
			}, {
				Name:       "udpdataport",
				Protocol:   "UDP",
				Port:       35002,
				TargetPort: intstr.IntOrString{intstr.Type(0), 35002, "35002"},
			}},
			Selector: map[string]string{"app": "netperfserver"},
		},
	}
	svcCr, errCr := clientset.CoreV1().Services(namespaceUDP).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if errCr != nil {
		panic(errCr)
	}
	fmt.Println("Service UDP my-service-netperf created " + svcCr.GetName())

	netSpeeds := make([]float64, iteration)
	confidenceArray := make([]float64, iteration)
	for i := 0; i < iteration; i++ {
		dep := createNetperfServer("15201", namespaceUDP)
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

		command := "netperf -t UDP_STREAM -H " + serviceC.Spec.ClusterIP + " -i 30,2 -p 15201 -v 2 -c -- -P ,35002 -D > file.txt; cat file.txt"
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
			podClient, errC = clientset.CoreV1().Pods(namespaceUDP).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfperfclient"})
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
					fmt.Println(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

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
	fmt.Printf("Test Namespace: %s deleted\n", namespaceUDP)
	avgSp, avgClient, avgServer := utils.AvgSpeed(netSpeeds, cpuC, cpuS, float64(iteration))
	return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + "/" + fmt.Sprintf("%f", avgServer)
}

func TCPHairpinservice(clientset *kubernetes.Clientset, multiple bool) string {

	svcCr := initializeTCPService(multiple, clientset, namespace, "netperfhairpin")
	netSpeeds := make([]float64, iteration)
	confidenceArray := make([]float64, iteration)
	for i := 0; i < iteration; i++ {
		serviceC, errSvcSearch := clientset.CoreV1().Services(namespace).Get(context.TODO(), svcCr.Name, metav1.GetOptions{})
		if errors.IsNotFound(errSvcSearch) {
			fmt.Printf("svc %s in namespace %s not found\n", svcCr.Name, namespace)
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
		command := "netserver -p 15001; sleep 10; netperf -H " + serviceC.Spec.ClusterIP + " -i 30,2 -j -p 15001 -v 2 -c -- -D -P ,35001> file.txt; cat file.txt"
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
						Name:   "netperfhairpin",
						Labels: map[string]string{"app": "netperfhairpin"},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:    "netperfserver",
								Image:   "leannet/k8s-netperf",
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
							},
						},
						RestartPolicy: "OnFailure",
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
		podHairpin, errC := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfhairpin"})
		if errC != nil {
			panic(errC)
		}
		for {
			if len(podHairpin.Items) != 0 {
				break
			}
			podHairpin, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfhairpin"})
			if errC != nil {
				panic(errC)
			}
		}
		fmt.Printf("Created pod %q.\n", podHairpin.Items[0].Name)
		pod := podHairpin.Items[0]
		var str string
		ctl := 0
		for ctl != 1 {
			switch pod.Status.Phase {
			case apiv1.PodRunning, apiv1.PodPending:
				{
					podHairpin, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfhairpin"})
					if errC != nil {
						panic(errC)
					}
					pod = podHairpin.Items[0]
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
					fmt.Println(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		velspeed, conf := calculateSpeed(str, clientset, namespace, 0)
		fmt.Printf("%d %f Gbits/sec \n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		netSpeeds[i] = velspeed
		confidenceArray[i] = conf
		cpuC[i] = float64(i)
		cpuS[i] = float64(i)
		errJobDel := clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{})
		if errJobDel != nil {
			panic(errJobDel)
		}

		JobSize, errWaitJobDel := clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
		if errWaitJobDel != nil {
			panic(errWaitJobDel)
		}
		for len(JobSize.Items) != 0 {
			JobSize, errWaitJobDel = clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
			if errWaitJobDel != nil {
				panic(errWaitJobDel)
			}
		}

		//Pod delete
		errPodDel := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		if errPodDel != nil {
			panic(errPodDel)
		}

		PodSize, errWaitPodDel := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
		if errWaitPodDel != nil {
			panic(errWaitPodDel)
		}
		for len(PodSize.Items) != 0 {
			PodSize, errWaitPodDel = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=netperfclient"})
			if errWaitPodDel != nil {
				panic(errWaitPodDel)
			}
		}

	}
	utils.DeleteBulk(10, clientset, namespace)
	utils.DeleteNS(clientset, namespace)
	avgSp, avgClient, avgServer := utils.AvgSpeed(netSpeeds, cpuC, cpuS, float64(iteration))
	return fmt.Sprintf("%f", avgSp) + " Gbits/sec, confidence avg" + fmt.Sprintf("%f", confidenceAVG(netSpeeds, confidenceArray, float64(iteration))) + " and cpu client/server usage : " + fmt.Sprintf("%f", avgClient) + "/" + fmt.Sprintf("%f", avgServer)
}

/*
func UDPHairpinservice(clientset *kubernetes.Clientset, multiple bool) string {

}*/

func initializeTCPService(multiple bool, clientset *kubernetes.Clientset, ns string, label string) *apiv1.Service {
	nsCr := utils.CreateNS(clientset, ns)
	fmt.Printf("Test Namespace: %s created\n", nsCr.GetName())

	if multiple {
		fmt.Println("the program will create multiple services and endpoints")
		utils.CreateBulk(10, clientset, namespace)
	}

	svc := apiv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService,
			Namespace: ns,
			//Labels: map[string]string{"":""},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:       "controltcp",
				Protocol:   "TCP",
				Port:       15001,
				TargetPort: intstr.IntOrString{intstr.Type(0), 15001, "15001"},
			},
				{
					Name:       "datatcp",
					Protocol:   "TCP",
					Port:       35001,
					TargetPort: intstr.IntOrString{intstr.Type(0), 35001, "35001"},
				}},
			Selector: map[string]string{"app": label},
		},
	}
	svcCr, errCr := clientset.CoreV1().Services(ns).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if errCr != nil {
		panic(errCr)
	}
	fmt.Println("Service my-service-netperf created " + svcCr.GetName())

	return svcCr
}
