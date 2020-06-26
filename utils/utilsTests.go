package utils

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"strconv"
)

func SetNodeSelector(casus int) string {
	var node string
	if casus == 1 {
		node = "node1"
	} else {
		node = "node2"
	}
	return node
}

func AvgSpeed(speeds []float64, cpuS []float64, cpuC []float64, div float64) (float64, float64, float64) {
	var max = 0.0
	var countM = -1
	var countm = -1
	var min = 10000.0
	for i := 0; i < int(div); i++ {
		if max < speeds[i] {
			max = speeds[i]
			countM = i
		}
		if min >= speeds[i] {
			min = speeds[i]
			countm = i
		}
	}
	fmt.Printf("\nvalore max: %f e valore min: %f\n", speeds[countM], speeds[countm])
	speeds[countM] = 0.0
	speeds[countm] = 0.0
	cpuC[countm] = 0.0
	cpuS[countm] = 0.0
	cpuC[countM] = 0.0
	cpuS[countM] = 0.0

	var sumSp, sumC, sumS float64
	sumSp = 0.0
	sumC = 0.0
	sumS = 0.0
	for i := 0; i < int(div); i++ {
		sumSp = sumSp + speeds[i]
		sumS = sumS + cpuS[i]
		sumC = sumC + cpuC[i]
	}
	div = div - 2
	return sumSp / div, sumS / div, sumC / div
}

func CreateNS(clientset *kubernetes.Clientset, ns string) *apiv1.Namespace {
	nsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	_, err1 := clientset.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	if err1 != nil {
		DeleteNS(clientset, ns)
		panic(err1)
	}

	return nsSpec
}

func DeleteNS(clientset *kubernetes.Clientset, ns string) {
	fmt.Println("Deleting namespace " + ns)
	errNs := clientset.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
	if errNs != nil {
		panic(errNs)
	}
	nsDel, errDel := clientset.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if errDel != nil {
		panic(errDel)
	}
	for nsDel != nil {
		nsDel, errDel = clientset.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
		if errDel != nil {
			nsDel = nil
		}
	}
}

func CreateBulk(numberServices int, clientset *kubernetes.Clientset, ns string) {
	/// this function creates a lot of services & endpoints , that aren't used, but can influence
	// throughput and CPU usage
	for i := 0; i < numberServices; i++ {
		createSvc(i, clientset, ns)
		createEndpoints(i, clientset, ns)
	}
}

func DeleteBulk(numberServices int, clientset *kubernetes.Clientset, ns string) {
	for i := 0; i < numberServices; i++ {
		deplName := "randomdepl" + strconv.Itoa(i)
		nameService := "randomservice" + strconv.Itoa(i)
		clientset.AppsV1().Deployments(ns).Delete(context.TODO(), deplName, metav1.DeleteOptions{})
		clientset.CoreV1().Services(ns).Delete(context.TODO(), nameService, metav1.DeleteOptions{})
	}
}

func createEndpoints(i int, clientset *kubernetes.Clientset, ns string) {
	deplName := "randomdepl" + strconv.Itoa(i)
	label := "casualserver" + strconv.Itoa(i)
	/*var p int32
	p = 20000 + int32(i)*/

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": label},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   deplName,
					Labels: map[string]string{"app": label},
				},

				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "nginx",
							Image:   "nginx:1.14.2",
							Command: []string{"/bin/sh"},
							Args:    []string{"-c", "tail -f /dev/null"},
						},
					},
				},
			},
		},
	}
	fmt.Println("Creating deployment...")
	res, errDepl := clientset.AppsV1().Deployments(ns).Create(context.TODO(), dep, metav1.CreateOptions{})
	if errDepl != nil {
		panic(errDepl)
	}
	fmt.Printf("Created deployment %q.\n", res.GetObjectMeta().GetName())
}

func createSvc(i int, clientset *kubernetes.Clientset, ns string) *apiv1.Service {
	nameService := "randomservice" + strconv.Itoa(i)
	label := "casualserver" + strconv.Itoa(i)

	var p int32
	p = 20000 + int32(i)
	svcRandom := apiv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService,
			Namespace: ns,
			//Labels: map[string]string{"":""},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:       "tcpport",
				Protocol:   "TCP",
				Port:       p,
				TargetPort: intstr.IntOrString{intstr.Type(0), 8080, "8080"},
			}},
			Selector: map[string]string{"app": label},
		},
	}
	svcCr, errCr := clientset.CoreV1().Services(ns).Create(context.TODO(), &svcRandom, metav1.CreateOptions{})
	if errCr != nil {
		panic(errCr)
	}
	fmt.Printf("Service %s created\n", svcCr.GetName())

	return svcCr
}

func CleanCluster(clientset *kubernetes.Clientset, ns string, labelServer string, labelClient string, deplName string, jobName string, podName string) {
	errDplDel := clientset.AppsV1().Deployments(ns).Delete(context.TODO(), deplName, metav1.DeleteOptions{})
	if errDplDel != nil {
		panic(errDplDel)
	}

	DeplSize, errWaitDeplDel := clientset.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelServer})
	if errWaitDeplDel != nil {
		panic(errWaitDeplDel)
	}
	for len(DeplSize.Items) != 0 {
		DeplSize, errWaitDeplDel = clientset.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelServer})
		if errWaitDeplDel != nil {
			panic(errWaitDeplDel)
		}
	}
	//wait until pod deply delete
	PodSize, errWaitPodSDel := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelServer})
	if errWaitPodSDel != nil {
		panic(errWaitPodSDel)
	}
	for len(PodSize.Items) != 0 {
		PodSize, errWaitPodSDel = clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelServer})
		if errWaitPodSDel != nil {
			panic(errWaitPodSDel)
		}
	}

	//Job delete
	errJobDel := clientset.BatchV1().Jobs(ns).Delete(context.TODO(), jobName, metav1.DeleteOptions{})
	if errJobDel != nil {
		panic(errJobDel)
	}
	JobSize, errWaitJobDel := clientset.BatchV1().Jobs(ns).List(context.TODO(), metav1.ListOptions{})
	if errWaitJobDel != nil {
		panic(errWaitJobDel)
	}
	for len(JobSize.Items) != 0 {
		JobSize, errWaitJobDel = clientset.BatchV1().Jobs(ns).List(context.TODO(), metav1.ListOptions{})
		if errWaitJobDel != nil {
			panic(errWaitJobDel)
		}
	}

	//Pod delete
	errPodDel := clientset.CoreV1().Pods(ns).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if errPodDel != nil {
		panic(errPodDel)
	}
	PodSize, errWaitPodDel := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelClient})
	if errWaitPodDel != nil {
		panic(errWaitPodDel)
	}
	for len(PodSize.Items) != 0 {
		PodSize, errWaitPodDel = clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelClient})
		if errWaitPodDel != nil {
			panic(errWaitPodDel)
		}
	}
}
