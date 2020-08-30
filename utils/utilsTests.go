package utils

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"strconv"
	"strings"
	"time"
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

func AvgSpeed(speeds []float64, cpuC []float64, cpuS []float64, confC []float64, confS []float64, div float64) (float64, float64, float64, float64, float64) {
	var sumSp, sumC, sumS, sumConfC, sumConfS float64
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

	if cpuC[0] != -100.00 {
		cpuC[countm] = 0.0
		cpuS[countm] = 0.0
		cpuC[countM] = 0.0
		cpuS[countM] = 0.0
		confC[countM] = 0.0
		confC[countm] = 0.0
		confS[countM] = 0.0
		confS[countm] = 0.0
		sumC = 0.0
		sumS = 0.0
		sumConfC = 0.0
		sumConfS = 0.0

	}
	sumSp = 0.0

	for i := 0; i < int(div); i++ {
		sumSp = sumSp + speeds[i]

		if cpuC[0] != -100.00 {
			sumS = sumS + cpuS[i]
			sumC = sumC + cpuC[i]
			sumConfC = sumConfC + confC[i]
			sumConfS = sumConfS + confS[i]
		}
	}
	div = div - 2
	if cpuC[0] != -100.00 {
		return sumSp / div, sumC / div, sumS / div, sumConfC / div, sumConfS / div
	}
	return sumSp / div, 0.0, 0.0, 0.0, 0.0
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

func CreateBulk(numberServices int, numberPods int, clientset *kubernetes.Clientset, ns string) {
	/// this function creates a lot of services & endpoints , that aren't used, but can influence
	// throughput and CPU usage
	/*for i := 0; i < numberPods; i++ {
		if numberServices != 0 {
			for j := 0; j < numberServices/numberPods; j++ {
				createSvc(j+((numberServices/numberPods)*i), i, clientset, ns)
			}
		}
		createPodsFake(i, clientset, ns)
	}*/
	for i := 0; i < numberPods; i++ {
		if numberServices != 0 {
			createSvc(i, i, clientset, ns)
		}
		createPodsFake(i, clientset, ns)
	}

}

func DeleteBulk(numberServices int, numberPods int, clientset *kubernetes.Clientset, ns string) {
	for i := 0; i < numberServices; i++ {
		nameService := "randomservice" + strconv.Itoa(i)
		clientset.CoreV1().Services(ns).Delete(context.TODO(), nameService, metav1.DeleteOptions{})
	}
	for i := 0; i < numberPods; i++ {
		deplName := "randomdepl" + strconv.Itoa(i)
		clientset.AppsV1().Deployments(ns).Delete(context.TODO(), deplName, metav1.DeleteOptions{})
	}
}

func CreateAllNetPol(clientset *kubernetes.Clientset, netpolNumber int, namespace string, labelServer string, labelClient string) string {
	// vado a bloccare il traffico diretto ai due pods
	networkPolicies := createNetPol(clientset, namespace, 1, labelServer, " ", "block-all-server")
	fmt.Printf("Created NetPol %s\n", networkPolicies.GetName())
	namePol := networkPolicies.GetName()

	networkPolicies = createNetPol(clientset, namespace, 1, labelClient, " ", "block-all-client")
	fmt.Printf("Created NetPol %s\n", networkPolicies.GetName())
	namePol = namePol + " " + networkPolicies.GetName()

	//permetto ai due pods di comunicare tra di loro
	networkPolicies = createNetPol(clientset, namespace, 2, labelServer, labelClient, "permit-iperf-server")
	namePol = namePol + " " + networkPolicies.GetName()
	fmt.Printf("Created NetPol %s\n", networkPolicies.GetName())
	networkPolicies = createNetPol(clientset, namespace, 2, labelClient, labelServer, "permit-iperf-client")
	namePol = namePol + " " + networkPolicies.GetName()
	fmt.Printf("Created NetPol %s\n", networkPolicies.GetName())

	//permetto agli altri pods di comunicare con pod1 e pod2
	for i := 0; i < netpolNumber; i++ {
		netPolName := "netpol-client-" + strconv.Itoa(i)
		label := "casualserver" + strconv.Itoa(i)
		networkPolicies = createNetPol(clientset, namespace, 2, labelServer, label, netPolName+"-server")
		namePol = namePol + " " + networkPolicies.GetName()
		fmt.Printf("Created NetPol %s\n", networkPolicies.GetName())
		networkPolicies = createNetPol(clientset, namespace, 2, labelClient, label, netPolName+"-client")
		namePol = namePol + " " + networkPolicies.GetName()
		fmt.Printf("Created NetPol %s\n", networkPolicies.GetName())
	}

	return namePol
}

func createPodsFake(i int, clientset *kubernetes.Clientset, ns string) {
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

func createSvc(j int, i int, clientset *kubernetes.Clientset, ns string) *apiv1.Service {
	nameService := "randomservice" + strconv.Itoa(j)
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
		DeleteNS(clientset, ns)
	}
	//fmt.Printf("Service %s created\n", svcCr.GetName())

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
	fmt.Printf("\n%d pod terminating\n", len(PodSize.Items))
	for len(PodSize.Items) != 0 {
		PodSize, errWaitPodSDel = clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelServer})
		if errWaitPodSDel != nil {
			panic(errWaitPodSDel)
		}
	}

	fmt.Println("arrivo qui con ancora degli item")

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
	time.Sleep(30 * time.Second)
}

func createNetPol(clientset *kubernetes.Clientset, namespace string, casus int, labelserver string, labelclient string, nameNetpol string) *v1.NetworkPolicy {
	var netPol *v1.NetworkPolicy
	if casus == 1 {
		netPol = &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: nameNetpol},
			Spec: v1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": labelserver,
					},
				},
				PolicyTypes: []v1.PolicyType{v1.PolicyTypeEgress, v1.PolicyTypeIngress},
				Ingress:     []v1.NetworkPolicyIngressRule{},
				Egress:      []v1.NetworkPolicyEgressRule{},
			},
		}
	} else {
		netPol = &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: nameNetpol},
			Spec: v1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": labelserver,
					},
				},
				PolicyTypes: []v1.PolicyType{v1.PolicyTypeEgress, v1.PolicyTypeIngress},
				Ingress: []v1.NetworkPolicyIngressRule{
					{
						From: []v1.NetworkPolicyPeer{{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": labelclient,
								},
							},
						}},
					},
				},
				Egress: []v1.NetworkPolicyEgressRule{
					{
						To: []v1.NetworkPolicyPeer{{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": labelclient,
								},
							},
						}},
					},
				},
			},
		}
	}
	networkpolicies, error := clientset.NetworkingV1().NetworkPolicies(namespace).Create(context.TODO(), netPol, metav1.CreateOptions{})
	if error != nil {
		fmt.Println("Errore nella creazione della policies")
		DeleteNS(clientset, namespace)
		panic(error)
	}
	return networkpolicies
}

func DeleteAllPolicies(clientset *kubernetes.Clientset, namespace string, namePol string) {
	allNames := strings.Split(namePol, " ")

	for _, singlePol := range allNames {
		fmt.Println("deleted netpol with name: " + singlePol)
		if err := clientset.NetworkingV1().NetworkPolicies(namespace).Delete(context.TODO(), singlePol, metav1.DeleteOptions{}); err != nil {
			DeleteNS(clientset, namespace)
			panic(err)
		}
	}
}
