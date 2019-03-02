/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
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

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d PVCs in the cluster\n", len(pvcs.Items))

	namespacesSeen := make(map[string]bool)
	for _, pvc := range pvcs.Items {
		if namespacesSeen[pvc.Namespace] {
			// Skip namespaces that we've already worked through
			fmt.Printf("Skipping namespace: %v\n", pvc.Namespace)
			continue
		}
		namespace, err := clientset.CoreV1().Namespaces().Get(pvc.Namespace, metav1.GetOptions{})
		// Check if namespace is not found
		// These are the PVCs that are wedged
		if err != nil && strings.Contains(err.Error(), "not found") {
			createNamespace(clientset, pvc.Namespace)
			deleteDeployments(clientset, pvc.Namespace)
			deleteStatefulSets(clientset, pvc.Namespace)
			deletePVCs(clientset, pvc.Namespace)
			deleteNamespace(clientset, pvc.Namespace)
			// Mark the namespace as seen
			namespacesSeen[pvc.Namespace] = true
		} else {
			fmt.Printf("Skipping PVC: %v:%v\n", namespace.Name, pvc.Name)
		}
	}
}

func createNamespace(clientset *kubernetes.Clientset, name string) {
	newNamespace := apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	fmt.Printf("Creating Namespace: %s\n", name)
	namespace, err := clientset.CoreV1().Namespaces().Create(&newNamespace)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Namespace created: %s\n", namespace.Name)
}

func deleteNamespace(clientset *kubernetes.Clientset, name string) {
	fmt.Printf("Deleting Namespace: %s\n", name)
	err := clientset.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Deleted namespace: %s\n", name)
}

func deleteDeployments(clientset *kubernetes.Clientset, namespace string) {
	fmt.Printf("Deleting deployments in namespace: %s\n", namespace)
	deployments, err := clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, deployment := range deployments.Items {
		fmt.Printf("Deleting v1Apps deployment: %s:%s\n", namespace, deployment.Name)
		err := clientset.AppsV1().Deployments(namespace).Delete(deployment.Name, &metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
	}
	deploymentsBeta, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
	for _, deployment := range deploymentsBeta.Items {
		fmt.Printf("Deleting v1Beta deployment: %s:%s\n", namespace, deployment.Name)
		err := clientset.ExtensionsV1beta1().Deployments(namespace).Delete(deployment.Name, &metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("Deleted deployments in namespace: %s\n", namespace)
}

func deleteStatefulSets(clientset *kubernetes.Clientset, namespace string) {
	fmt.Printf("Deleting statefulsets in namespace: %s\n", namespace)
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, statefulset := range statefulsets.Items {
		fmt.Printf("Deleting v1Apps statefulset: %s:%s\n", namespace, statefulset.Name)
		err := clientset.AppsV1().StatefulSets(namespace).Delete(statefulset.Name, &metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("Deleted statefulsets in namespace: %s\n", namespace)
}

func deletePVCs(clientset *kubernetes.Clientset, namespace string) {
	fmt.Printf("Deleting PVC in namespace: %v\n", namespace)
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, pvc := range pvcs.Items {
		fmt.Printf("Deleting v1Apps pvc: %s:%s\n", namespace, pvc.Name)
		err := clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(pvc.Name, &metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("Deleted PVC: %v\n", namespace)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
