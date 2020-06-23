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

func AvgSpeed(speeds [12]float64) float64 {
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

func CreateNS(clientset *kubernetes.Clientset, ns string) *apiv1.Namespace {
	nsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	_, err1 := clientset.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	if err1 != nil {
		panic(err1)
	}

	return nsSpec
}

func DeleteNS(clientset *kubernetes.Clientset, namespace string) {
	errNs := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if errNs != nil {
		panic(errNs)
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
