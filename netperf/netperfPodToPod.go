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
	appString "strings"
	iperfPTP "testk8s/iperf"
	"time"
)

var deplName = "servernetperf"
var namespace = "testnetperf"
var jobName = "jobnetperfclient"

func NetperfTCPPodtoPod(clientset *kubernetes.Clientset) string {

	nsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err1 := clientset.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	if err1 != nil {
		panic(err1)
	}

	fmt.Println("Namespace testnetperf created")

	//create one deployment of netperf server

	var netSpeeds [12]float64

	for i := 0; i < 12; i++ {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: namespace,
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
								Args:    []string{"-c", "netserver -p 15001 -v 2 -d; tail -f /dev/null"},
							},
						},
						NodeSelector: map[string]string{"type": "node2"},
					},
				},
			},
		}
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
		command := "netperf -H " + podI.Status.PodIP + " -i 30,2 -j -p 15001 -v 2  -- -D> file.txt; cat file.txt"
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
						NodeSelector:  map[string]string{"type": "node1"},
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
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		vectString := appString.Split(str, "\n")
		strspeed := appString.Split(vectString[6], "    ")
		strspeed = appString.Split(strspeed[2], " ")
		fmt.Println(strspeed[0])
		speed := "Mbits/sec"
		velspeed, errConv := strconv.ParseFloat(strspeed[0], 32)
		if errConv != nil {
			panic(errConv)
		}

		switch speed {
		case "Mbits/sec":
			velspeed = velspeed / 1000
		case "Kbits/sec":
			velspeed = velspeed / 1000000
		case "Gbits/sec":
			fmt.Println("Ok, Gbits/sec")

		}

		fmt.Printf("%d %f Gbits/sec \n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		netSpeeds[i] = velspeed

		//Deployment delete

		errDplDel := clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), deplName, metav1.DeleteOptions{})
		if errDplDel != nil {
			panic(errDplDel)
		}
		DeplSize, errWaitDeplDel := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if errWaitDeplDel != nil {
			panic(errWaitDeplDel)
		}
		for len(DeplSize.Items) != 0 {
			DeplSize, errWaitDeplDel = clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
			if errWaitDeplDel != nil {
				panic(errWaitDeplDel)
			}
		}

		//Job delete

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

		PodSize, errWaitPodDel := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if errWaitPodDel != nil {
			panic(errWaitPodDel)
		}
		for len(PodSize.Items) != 0 {
			PodSize, errWaitPodDel = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if errWaitPodDel != nil {
				panic(errWaitPodDel)
			}
		}
	}
	errNs := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})

	if errNs != nil {
		panic(errNs)
	}
	fmt.Printf("Test Namespace: %s deleted\n", namespace)
	time.Sleep(10 * time.Second)
	return fmt.Sprintf("%f", iperfPTP.AvgSpeed(netSpeeds)) + " Gbits/sec"
}

func NetperfUDPPodtoPod(clientset *kubernetes.Clientset) string {

	nsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err1 := clientset.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	if err1 != nil {
		panic(err1)
	}

	fmt.Println("Namespace UDP testnetperf created")

	//create one deployment of netperf server UDP

	var netSpeeds [12]float64

	for i := 0; i < 12; i++ {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: namespace,
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
								Args:    []string{"-c", "netserver -p 15003 -v 2 -d; tail -f /dev/null"},
							},
						},
						NodeSelector: map[string]string{"type": "node2"},
					},
				},
			},
		}
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
				pod, namespace, statusError.ErrStatus.Message)
		} else if errPodSearch != nil {
			panic(errPodSearch.Error())
		} else {
			fmt.Printf("Found pod %s in namespace %s\n", pod.Name, namespace)
			podIP := podI.Status.PodIP
			fmt.Printf("UDP Server IP: %s\n", podIP)

		}

		command := "netperf -t UDP_STREAM -H " + podI.Status.PodIP + " -i 30,2 -p 15003 -v 2  -- -R 1 -D > file.txt; cat file.txt"
		fmt.Println("Creating UDP Netperf Client: " + command)
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
						NodeSelector:  map[string]string{"type": "node1"},
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
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		fmt.Println(str)
		vectString := appString.Split(str, "\n")
		strspeed := appString.Split(vectString[5], " ")
		fmt.Println(strspeed[len(strspeed)-1])
		speed := "Mbits/sec"
		velspeed, errConv := strconv.ParseFloat(strspeed[len(strspeed)-1], 32)
		if errConv != nil {
			panic(errConv)
		}

		switch speed {
		case "Mbits/sec":
			velspeed = velspeed / 1000
		case "Kbits/sec":
			velspeed = velspeed / 1000000
		case "Gbits/sec":
			fmt.Println("Ok, Gbits/sec")

		}

		fmt.Printf("%d %f Gbits/sec \n", i, velspeed)
		//todo vedere cosa succede con float 32, per ora 64
		netSpeeds[i] = velspeed

		//Deployment delete

		errDplDel := clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), deplName, metav1.DeleteOptions{})
		if errDplDel != nil {
			panic(errDplDel)
		}
		DeplSize, errWaitDeplDel := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if errWaitDeplDel != nil {
			panic(errWaitDeplDel)
		}
		for len(DeplSize.Items) != 0 {
			DeplSize, errWaitDeplDel = clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
			if errWaitDeplDel != nil {
				panic(errWaitDeplDel)
			}
		}

		//Job delete

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

		PodSize, errWaitPodDel := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if errWaitPodDel != nil {
			panic(errWaitPodDel)
		}
		for len(PodSize.Items) != 0 {
			PodSize, errWaitPodDel = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if errWaitPodDel != nil {
				panic(errWaitPodDel)
			}
		}
	}
	errNs := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})

	if errNs != nil {
		panic(errNs)
	}
	fmt.Printf("UDP Test Namespace: %s deleted\n", namespace)
	time.Sleep(10 * time.Second)
	return fmt.Sprintf("%f", iperfPTP.AvgSpeed(netSpeeds)) + " Gbits/sec"
}
