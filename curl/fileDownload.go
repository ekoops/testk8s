package curl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"os"
	"strconv"
	"strings"
	"testk8s/utils"
	"time"
)

var node = ""
var node2 = "node2"
var iteration = 6

func SpeedMovingFileandLatency(clientset *kubernetes.Clientset, numberReplicas int, casus int, fileoutput *os.File, servicesNumber int) string {

	node = utils.SetNodeSelector(casus)

	namespace := "namespacecurl" + strconv.Itoa(casus)
	ns := utils.CreateNS(clientset, namespace)
	netSpeeds := make([]float64, iteration)
	netLatency := make([]float64, iteration)
	println("creato namespace " + ns.GetName())
	svcCr := createCurlService(clientset, "mycurlservice", namespace, "mycurl")
	println("creato service " + svcCr.GetName())

	if servicesNumber > 1 {
		utils.CreateBulk(servicesNumber, servicesNumber, clientset, namespace)
	}

	dep := createCurlDeployment(namespace, numberReplicas, "curlserver")
	fmt.Println("Creating deployment...")
	res, errDepl := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), dep, metav1.CreateOptions{})
	if errDepl != nil {
		utils.DeleteNS(clientset, namespace)
		panic(errDepl)
	}
	fmt.Printf("Created deployment %q.\n", res.GetObjectMeta().GetName())
	fmt.Println(time.Now())
	podvect, errP := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=mycurl"})
	if errP != nil {
		utils.DeleteNS(clientset, namespace)
		panic(errP)
	}
	var num = len(podvect.Items)
	fmt.Printf("Wait for pod creation.. %d\n", num)
	for num < numberReplicas {
		podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=mycurl"})
		num = len(podvect.Items)
		if errP != nil {
			utils.DeleteNS(clientset, namespace)
			panic(errP)
		}
		fmt.Print(".")
	}
	fmt.Printf("There are %d pods in the cluster\n", len(podvect.Items))

	lungh := len(podvect.Items)
	for i := 0; i < lungh; i++ {
		pod := podvect.Items[i]
		ctl := 0
		for ctl != 1 {
			switch pod.Status.Phase {
			case apiv1.PodRunning:
				{
					ctl = 1
					fmt.Printf("\n pod %s è Running %d \n", pod.GetName(), i)
					break
				}
			case apiv1.PodPending:
				{
					podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=mycurl"})
					lungh = len(podvect.Items)
					if errP != nil {
						utils.DeleteNS(clientset, namespace)
						panic(errP)
					}
					pod = podvect.Items[i]
				}
			case apiv1.PodFailed, apiv1.PodSucceeded:
				{
					utils.DeleteNS(clientset, namespace)
					panic("error in pod creation")
				}
			}
		}
	}
	redo := true
	for redo {
		fmt.Printf("tutti i deployment dovrebbero essere running")
		fmt.Println(time.Now())
		command := "for i in 0 1 2 3 4 5 ; do curl http://" + svcCr.Spec.ClusterIP + ":8080 -o dev/null -w \"TTFB: %{time_starttransfer} \" >> file.txt; done ;cat file.txt"
		jobsClient := clientset.BatchV1().Jobs(namespace)
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "clientcurl",
				Namespace: namespace,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit: pointer.Int32Ptr(4),
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "clientcurl",
						Labels: map[string]string{"app": "curlclient"},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:    "curlclient",
								Image:   "nginx",
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
			utils.DeleteNS(clientset, namespace)
			panic(errJ)
		}

		fmt.Println(time.Now())
		fmt.Printf("Created job %q.\n", result1.GetObjectMeta().GetName())
		podClient, errC := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curlclient"})
		if errC != nil {
			utils.DeleteNS(clientset, namespace)
			panic(errC)
		}
		for {
			if len(podClient.Items) != 0 {
				break
			}
			podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curlclient"})
			if errC != nil {
				utils.DeleteNS(clientset, namespace)
				panic(errC)
			}
		}
		fmt.Printf("Created pod %q.\n", podClient.Items[0].Name)
		fmt.Println(time.Now())

		podC := podClient.Items[0]
		var str string
		ctl := 0
		for ctl != 1 {
			switch podC.Status.Phase {
			case apiv1.PodRunning, apiv1.PodPending:
				{
					podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curlclient"})
					if errC != nil {
						utils.DeleteNS(clientset, namespace)
						panic(errC)
					}
					podC = podClient.Items[0]
				}
			case apiv1.PodSucceeded:
				{
					logs := clientset.CoreV1().Pods(namespace).GetLogs(podC.Name, &apiv1.PodLogOptions{})
					podLogs, errLogs := logs.Stream(context.TODO())
					if errLogs != nil {
						utils.DeleteNS(clientset, namespace)
						panic(errLogs)
					}
					defer podLogs.Close()
					buf := new(bytes.Buffer)
					_, errBuf := io.Copy(buf, podLogs)
					if errBuf != nil {
						utils.DeleteNS(clientset, namespace)
						panic(errBuf)
					}
					str = buf.String()
					if strings.Contains(str, " Failed to connect ") {
						redo = true
						fmt.Printf("Errore di connessione al svc")
						utils.CleanCluster(clientset, namespace, "", "curlclient", "", job.GetName(), podC.GetName())
						continue
					} else {
						redo = false
					}
					fileoutput.WriteString(str)
					ctl = 1
					break
				}
			case apiv1.PodFailed:
				{
					utils.DeleteNS(clientset, namespace)
					panic("error panic in pod created by job")
				}

			}
		}
		netSpeeds, netLatency = calculateResults(str, clientset, namespace)
		utils.CleanCluster(clientset, namespace, "app=mycurl", "curlclient", dep.GetName(), job.GetName(), podC.GetName())
	}

	if servicesNumber > 1 {
		utils.DeleteBulk(servicesNumber, servicesNumber, clientset, namespace)
	}

	utils.DeleteNS(clientset, namespace)
	return speedAVG(netSpeeds, netLatency)
}

func calculateResults(str string, clientset *kubernetes.Clientset, namespace string) ([]float64, []float64) {
	var maxPos = -1
	var minPos = -1
	var maxPosLat = -1
	var minPosLat = -1
	var maxSpeed = -10.0
	var minSpeed = 10000.0
	var maxLatency = -10.0
	var minLatency = 10000.0
	var errLatencyConv error
	netSpeeds := make([]float64, iteration)
	netLatency := make([]float64, iteration)

	subLatency := strings.Split(str, "TTFB: ")
	for i := 1; i < len(subLatency); i++ {

		//latencyString := strings.Split(subLatency[len(subLatency)-1], ":")[len(strings.Split(subLatency[len(subLatency)-1], ":"))-1]
		subLatency[i] = strings.ReplaceAll(subLatency[i], " ", "")
		netLatency[i-1], errLatencyConv = strconv.ParseFloat(subLatency[i], 64)

		if netLatency[i-1] > maxLatency {
			fmt.Printf("\naggiorno massima latenza all'%d con %f\n", i-1, netLatency[i-1])
			maxPosLat = i - 1
			maxLatency = netLatency[i-1]
		}
		if netLatency[i-1] <= minLatency {
			fmt.Printf("\naggiorno minima latenza all'%d con %f\n", i-1, netLatency[i-1])
			minPosLat = i - 1
			minLatency = netLatency[i-1]
		}

		fmt.Printf("\n%f latenza\n", netLatency[i-1])
		if errLatencyConv != nil {
			fmt.Println("Errore nel speed conversion line 216")
			utils.DeleteNS(clientset, namespace)
			panic(errLatencyConv)
		}
	}

	substring := strings.Split(subLatency[0], "  % Total ")
	for i := 1; i < len(substring); i++ {
		fmt.Printf("\n valori attuali: %f %f %f %f\n", minSpeed, minLatency, maxSpeed, maxLatency)
		vectString := strings.Split(substring[i], "\r")

		vectString = strings.Split(vectString[len(vectString)-1], " 0 ")
		vectString[len(vectString)-2] = strings.ReplaceAll(vectString[len(vectString)-2], " ", "")
		fmt.Printf("%s\n", vectString[len(vectString)-2])
		switch vectString[len(vectString)-2][len(vectString[len(vectString)-2])-1] {
		case 'M':
			{
				vectString[len(vectString)-2] = strings.Replace(vectString[len(vectString)-2], "M", "", 2)
				speed, errConv := strconv.ParseFloat(vectString[len(vectString)-2], 64)
				if errConv != nil {
					fmt.Println("Errore nel speed conversion line Mega")
					utils.DeleteNS(clientset, namespace)
					panic(errConv)
				}
				speed = speed / 1000
				if speed > maxSpeed {
					fmt.Printf("\naggiorno massima speed all'%d con %f\n", i, speed)
					maxSpeed = speed
					maxPos = i - 1
				}
				if speed <= minSpeed {
					fmt.Printf("\naggiorno min speed all'%d con %f\n", i, speed)
					minSpeed = speed
					minPos = i - 1
				}
				netSpeeds[i-1] = speed
			}
		case 'K':
			{
				vectString[len(vectString)-2] = strings.Replace(vectString[len(vectString)-2], "K", "", 2)
				speed, errConv := strconv.ParseFloat(vectString[len(vectString)-2], 64)
				if errConv != nil {
					fmt.Println("Errore nel speed conversion line Kylo")
					utils.DeleteNS(clientset, namespace)
					panic(errConv)
				}
				speed = speed / 1000000
				if speed > maxSpeed {
					maxSpeed = speed
					maxPos = i - 1
				}
				if speed <= minSpeed {
					minSpeed = speed
					minPos = i - 1
				}
				netSpeeds[i-1] = speed
			}
		case 'G':
			{
				vectString[len(vectString)-2] = strings.Replace(vectString[len(vectString)-2], "G", "", 2)
				speed, errConv := strconv.ParseFloat(vectString[len(vectString)-2], 64)
				if errConv != nil {
					fmt.Println("Errore nel speed conversion line Giga")
					utils.DeleteNS(clientset, namespace)
					panic(errConv)
				}
				if speed > maxSpeed {
					maxSpeed = speed
					maxPos = i - 1
				}
				if speed <= minSpeed {
					minSpeed = speed
					minPos = i - 1
				}
				netSpeeds[i-1] = speed
			}

		}
	}

	fmt.Printf("\n %d max pos: %f\n", maxPos, netSpeeds[maxPos])
	netSpeeds[maxPos] = 0.0
	fmt.Println("\nSono arrivato qui e mi blocco maxLAtency")
	fmt.Printf("%d:%f max \n", maxPosLat, netLatency[maxPosLat])
	netLatency[maxPosLat] = 0.0
	fmt.Printf("%d:%f min \n", minPos, netSpeeds[minPos])
	netSpeeds[minPos] = 0.0
	fmt.Println("\nSono arrivato qui e mi blocco minLAtency")
	fmt.Printf("%d:%f min \n", minPosLat, netLatency[minPosLat])
	netLatency[minPosLat] = 0.0
	return netSpeeds, netLatency
}

func speedAVG(speeds []float64, latencies []float64) string {
	var sumSpeed = 0.0
	var sumLatency = 0.0

	for i := 0; i < iteration; i++ {
		sumSpeed = sumSpeed + speeds[i]
		sumLatency = sumLatency + latencies[i]
	}
	div := iteration - 2
	speed := fmt.Sprintf("%f", sumSpeed/float64(div))
	latency := fmt.Sprintf("%f", sumLatency/float64(div))
	return speed + " and latency is: " + latency
}

func createCurlDeployment(namespace string, replicas int, deplName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(int32(replicas)), /**/
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "mycurl"},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "mycurl",
					Labels: map[string]string{"app": "mycurl"},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
							Ports: []apiv1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "workdir",
									MountPath: "/usr/share/nginx/html",
								},
							},
						},
					},
					InitContainers: []apiv1.Container{
						{
							Name:  "install",
							Image: "busybox",
							Command: []string{
								"/bin/sh", "-c", "dd if=/dev/zero of=/work-dir/index.html bs=1024 count=102400",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "workdir",
									MountPath: "/work-dir",
								},
							},
						},
					},
					NodeSelector: map[string]string{"type": node2},
					Volumes: []apiv1.Volume{
						{
							Name: "workdir",
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{
									Medium:    "",
									SizeLimit: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func createCurlService(clientset *kubernetes.Clientset, nameService string, ns string, label string) *apiv1.Service {
	svc := apiv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService,
			Namespace: ns,
			//Labels: map[string]string{"":""},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.IntOrString{intstr.Type(0), 80, "80"},
			}},
			Selector: map[string]string{"app": label},
		},
	}
	svcCr, errCr := clientset.CoreV1().Services(ns).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if errCr != nil {
		utils.DeleteNS(clientset, ns)
		panic(errCr)
	}
	fmt.Println("Service: " + svcCr.GetName() + " created")
	return svcCr
}
