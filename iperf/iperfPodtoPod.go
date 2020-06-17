package iperf

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	"os"
	"path/filepath"
	"strconv"
	appString "strings"
	"time"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

var deplName = "serveriperf3"
var namespace = "testiperf"
var jobName = "jobiperfclient"

func IperfTCPPodtoPod() string {
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

	nsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err1 := clientset.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	if err1 != nil {
		panic(err)
	}

	fmt.Println("Namespace testiperf created")

	//create one deployment of iperf server

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
								Image:   "mlabbe/iperf3",
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", "iperf3 -s -p 5002 -d -V "},
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
					podvect, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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

		time.Sleep(10 * time.Second)
		command := "iperf3 -c " + podI.Status.PodIP + " -p 5002 -V -N -t 10 -Z > file.txt; cat file.txt"
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
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		vectString := appString.Split(str, "0.00-10.00 ")
		substringSpeed := appString.Split(vectString[1], "  ")
		speed := appString.Split(substringSpeed[2], " ")
		velspeed, errConv := strconv.ParseFloat(speed[0], 32)
		if errConv != nil {
			panic(errConv)
		}

		switch speed[1] {
		case "Mbits/sec":
			velspeed = velspeed / 1000
		case "Kbits/sec":
			velspeed = velspeed / 1000000
		case "Gbits/sec":
			fmt.Println("Ok, Gbits/sec")
		}

		fmt.Printf("%d %f %s \n", i, velspeed, speed[1])
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
	return fmt.Sprintf("%f", avgSpeed(netSpeeds)) + " Gbits/sec"
}

func IperfUDPPodtoPod() string {
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

	nsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err1 := clientset.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	if err1 != nil {
		panic(err)
	}

	fmt.Println("Namespace UDP testiperf created")

	//create one deployment of iperf server UDP

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
								Image:   "mlabbe/iperf3",
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", "iperf3 -s -p 5003 -d -V "},
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
					podvect, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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

		time.Sleep(10 * time.Second)
		command := "iperf3 -c " + podI.Status.PodIP + " -u -b 0 -p 5003 -V -N -t 10 -Z > file.txt; cat file.txt"
		fmt.Println("Creating UDP Iperf Client: " + command)
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
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				panic("error panic in pod created by job")
			}
		}

		//works on strings
		vectString := appString.Split(str, "0.00-10.00 ")
		substringSpeed := appString.Split(vectString[1], "  ")
		speed := appString.Split(substringSpeed[2], " ")
		velspeed, errConv := strconv.ParseFloat(speed[0], 32)
		if errConv != nil {
			panic(errConv)
		}

		switch speed[1] {
		case "Mbits/sec":
			velspeed = velspeed / 1000
		case "Kbits/sec":
			velspeed = velspeed / 1000000
		case "Gbits/sec":
			fmt.Println("Ok, Gbits/sec")
		}

		fmt.Printf("%d %f %s \n", i, velspeed, speed[1])
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
	return fmt.Sprintf("%f", avgSpeed(netSpeeds)) + " Gbits/sec"
}

func avgSpeed(speeds [12]float64) float64 {
	var max = 0.0
	var countM = -1
	var countm = -1
	var min = 10000.0
	for i := 0; i < 12; i++ {
		if max < speeds[i] {
			max = speeds[i]
			countM = i
		}
		if min >= speeds[i] {
			min = speeds[i]
			countm = i
		}
	}
	speeds[countM] = 0.0
	speeds[countm] = 0.0
	var sum = 0.0
	for i := 0; i < 12; i++ {
		sum = sum + speeds[i]
	}

	return sum / 10
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
