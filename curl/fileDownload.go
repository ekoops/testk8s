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
var iteration = 12
var max = 0.0
var min = 10000.0

func SpeedMovingFile(clientset *kubernetes.Clientset, numberReplicas int, casus int, fileoutput *os.File) string {

	node = utils.SetNodeSelector(casus)
	var maxPos = -1
	var minPos = -1
	namespace := "namespacecurl" + strconv.Itoa(casus)
	ns := utils.CreateNS(clientset, namespace)
	netSpeeds := make([]float64, iteration)
	println("creato namespace " + ns.GetName())
	svcCr := createCurlService(clientset, "mycurlservice", namespace, "mycurl")
	println("creato service " + svcCr.GetName())

	for i := 0; i < iteration; i++ {
		dep := createCurlDeployment(namespace, numberReplicas, "curlserver")
		fmt.Println("Creating deployment...")
		res, errDepl := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), dep, metav1.CreateOptions{})
		if errDepl != nil {
			panic(errDepl)
		}
		fmt.Printf("Created deployment %q.\n", res.GetObjectMeta().GetName())

		time.Sleep(10 * time.Second)

		podvect, errP := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=mycurl"})
		if errP != nil {
			panic(errP)
		}
		fmt.Print("Wait for pod creation..")
		for {
			podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=mycurl"})
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
		fmt.Printf("There are %d pods in the cluster\n", len(podvect.Items))
		for i := 0; i < len(podvect.Items); i++ {
			pod := podvect.Items[i]
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
						podvect, errP = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=mycurl"})
						if errP != nil {
							panic(errP)
						}
						pod = podvect.Items[i]
					}
				case apiv1.PodFailed, apiv1.PodSucceeded:
					panic("error in pod creation")
				}
			}
		}
		fmt.Printf("tutti i deployment dovrebbero essere running")
		command := "apt-get update -y; apt-get install curl -y; curl http://" + svcCr.Spec.ClusterIP + ":8080 -o dev/null >> file.txt;cat file.txt"

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
			panic(errJ)
		}
		fmt.Printf("Created job %q.\n", result1.GetObjectMeta().GetName())
		podClient, errC := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curlclient"})
		if errC != nil {
			panic(errC)
		}
		for {
			if len(podClient.Items) != 0 {
				break
			}
			podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curlclient"})
			if errC != nil {
				panic(errC)
			}
		}
		fmt.Printf("Created pod %q.\n", podClient.Items[0].Name)

		podC := podClient.Items[0]
		var str string
		ctl := 0
		for ctl != 1 {
			switch podC.Status.Phase {
			case apiv1.PodRunning, apiv1.PodPending:
				{
					podClient, errC = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curlclient"})
					if errC != nil {
						panic(errC)
					}
					podC = podClient.Items[0]
				}
			case apiv1.PodSucceeded:
				{
					logs := clientset.CoreV1().Pods(namespace).GetLogs(podC.Name, &apiv1.PodLogOptions{})
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

		vectString := strings.Split(str, "\n")
		fmt.Printf("%s", vectString[len(vectString)-1])
		vectString = strings.Split(vectString[len(vectString)-2], "0")
		vectString[len(vectString)-2] = strings.ReplaceAll(vectString[len(vectString)-2], " ", "")
		switch vectString[len(vectString)-2][len(vectString[len(vectString)-2])-1] {
		case 'M':
			{
				vectString[len(vectString)-2] = strings.Replace(vectString[len(vectString)-2], "M", "0", 2)
				speed, errConv := strconv.ParseFloat(vectString[len(vectString)-2], 64)
				if errConv != nil {
					fmt.Println("Errore nel speed conversion line Mega")
					panic(errConv)
				}
				speed = speed / 1000
				if speed > max {
					max = speed
					maxPos = i
				}
				if speed <= min {
					min = speed
					minPos = i
				}
				netSpeeds[i] = speed
			}
		case 'K':
			{
				vectString[len(vectString)-2] = strings.Replace(vectString[len(vectString)-2], "M", "0", 2)
				speed, errConv := strconv.ParseFloat(vectString[len(vectString)-2], 64)
				if errConv != nil {
					fmt.Println("Errore nel speed conversion line Kylo")
					panic(errConv)
				}
				speed = speed / 1000000
				if speed > max {
					max = speed
					maxPos = i
				}
				if speed <= min {
					min = speed
					minPos = i
				}
				netSpeeds[i] = speed
			}
		case 'G':
			{
				vectString[len(vectString)-2] = strings.Replace(vectString[len(vectString)-2], "M", "0", 2)
				speed, errConv := strconv.ParseFloat(vectString[len(vectString)-2], 64)
				if errConv != nil {
					fmt.Println("Errore nel speed conversion line Giga")
					panic(errConv)
				}
				if speed > max {
					max = speed
					maxPos = i
				}
				if speed <= min {
					min = speed
					minPos = i
				}
				netSpeeds[i] = speed
			}
		}

		utils.CleanCluster(clientset, namespace, "mycurl", "curlclient", dep.GetName(), job.GetName(), podC.GetName())
	}

	netSpeeds[maxPos] = 0.0
	netSpeeds[minPos] = 0.0

	return speedAVG(netSpeeds)
}

func speedAVG(speeds []float64) string {
	var sum = 0.0

	for i := 0; i < iteration; i++ {
		sum = sum + speeds[i]
	}
	s := fmt.Sprintf("%f", sum/10)
	return s
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
								"wget", "-O", "/work-dir/index.html", "http://speedtest.tele2.net/1GB.zip", /*http://speedtest.tele2.net/100MB.zip*/
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
		panic(errCr)
	}
	fmt.Println("Service my-service-iperf created " + svcCr.GetName())
	return svcCr
}
