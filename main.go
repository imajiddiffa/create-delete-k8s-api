// main.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
)

// Response - Our struct for all res[pmse]
type (
	PodParam struct {
		PodName   string `json:"pod_name"`
		PodIP     string `json:"pod_ip"`
		Namespace string `json:"namespace"`
		Image     string `json:"image"`
		Port      string `json:"port"`
	}

	Resource struct {
		APIVersion string                   `json:"apiVersion"`
		Items      []map[string]interface{} `json:"items"`
		Kind       string                   `json:"kind"`
		Metadata   map[string]interface{}   `json:"metadata"`
	}

	Pod struct {
		PodName        string `json:"pod_name"`
		PodIP          string `json:"pod_ip"`
		NodeName       string `json:"node_name"`
		NodeIP         string `json:"node_ip"`
		Namespace      string `json:"namespace"`
		ServiceAccount string `json:"service_account"`
	}
)

func createNewPod(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var podParam PodParam
	json.Unmarshal(reqBody, &podParam)

	// get pod template & convert to string
	podTemplate, err := ioutil.ReadFile("templates/pod.yml")
	if err != nil {
		log.Println("Err: Failed To Load Template")
		return
	}
	strpodTemplate := string(podTemplate)

	// replace template variables with values from parameters
	strpodTemplate = strings.Replace(strpodTemplate, "$POD_NAME", podParam.PodName, -1)
	strpodTemplate = strings.Replace(strpodTemplate, "$NAMESPACE", podParam.Namespace, -1)
	strpodTemplate = strings.Replace(strpodTemplate, "$POD_IP", podParam.PodIP, -1)
	strpodTemplate = strings.Replace(strpodTemplate, "$IMAGE", podParam.Image, -1)
	strpodTemplate = strings.Replace(strpodTemplate, "$PORT", podParam.Port, -1)

	// write prepared template to temp file
	err = ioutil.WriteFile("templates/temp.yml", []byte(strpodTemplate), 0644)
	if err != nil {
		log.Println("Err: Failed To Write Temp Template")
		return
	}

	// create namespace
	respNamespace, err := exec.Command("kubectl", "create", "namespace", podParam.Namespace).Output()
	if err != nil {
		strRespNamespace := string(respNamespace)
		log.Println(strRespNamespace)
		log.Println("Err: Failed To Create Namespace ", strRespNamespace)
	}

	// apply temp yaml
	respPod, err := exec.Command("kubectl", "apply", "-f", "templates/temp.yml").Output()
	if err != nil {
		strRespPod := string(respPod)
		log.Println(strRespPod)
		log.Println("Err: Failed To Create Pod ", strRespPod)
		return
	}
	strRespPod := string(respPod)
	log.Println(strRespPod)

	resp := map[string]interface{}{
		"data": strRespPod,
		"code": http.StatusOK,
	}

	json.NewEncoder(w).Encode(resp)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
}

func returnAllPods(w http.ResponseWriter, r *http.Request) {
	var resource Resource
	var pods []Pod
	var namespace string

	namespaceParam, ok := r.URL.Query()["namespace"]
	if ok {
		namespace = "-n=" + namespaceParam[0]
	}

	respPods, err := exec.Command("kubectl", "get", "pods", namespace, "-o=json").Output()
	if err != nil {
		log.Println("Err: ", err)
		json.NewEncoder(w).Encode(err)
	}

	err = json.Unmarshal(respPods, &resource)
	if err != nil {
		log.Println("Err: ", err)
		json.NewEncoder(w).Encode(err)
	}

	for _, v := range resource.Items {
		metadata, _ := v["metadata"]
		md, _ := metadata.(map[string]interface{})
		spec, _ := v["spec"]
		sp, _ := spec.(map[string]interface{})
		status, _ := v["status"]
		st, _ := status.(map[string]interface{})

		pod := Pod{
			PodName:        md["name"].(string),
			PodIP:          st["podIP"].(string),
			NodeName:       sp["nodeName"].(string),
			NodeIP:         st["hostIP"].(string),
			Namespace:      md["namespace"].(string),
			ServiceAccount: sp["serviceAccountName"].(string),
		}
		pods = append(pods, pod)
	}

	resp := map[string]interface{}{
		"data": pods,
		"code": http.StatusOK,
	}

	json.NewEncoder(w).Encode(resp)
}

func returnSinglePod(w http.ResponseWriter, r *http.Request) {
	var resource map[string]interface{}
	var namespace string
	vars := mux.Vars(r)
	name := vars["name"]

	namespaceParam, ok := r.URL.Query()["namespace"]
	if ok {
		namespace = "-n=" + namespaceParam[0]
	}

	respPod, err := exec.Command("kubectl", "get", "pods", name, namespace, "-o=json").Output()
	if err != nil {
		log.Println("Err: ", err)
		json.NewEncoder(w).Encode(err)
	}

	strResp := string(respPod)
	if len(strResp) == 0 {
		log.Println("Err: Pod Not Found")
		return
	}

	err = json.Unmarshal(respPod, &resource)
	if err != nil {
		log.Println("Err: ", err)
		json.NewEncoder(w).Encode(err)
	}

	metadata, _ := resource["metadata"]
	md, _ := metadata.(map[string]interface{})
	spec, _ := resource["spec"]
	sp, _ := spec.(map[string]interface{})
	status, _ := resource["status"]
	st, _ := status.(map[string]interface{})

	pod := Pod{
		PodName:        md["name"].(string),
		PodIP:          st["podIP"].(string),
		NodeName:       sp["nodeName"].(string),
		NodeIP:         st["hostIP"].(string),
		Namespace:      md["namespace"].(string),
		ServiceAccount: sp["serviceAccountName"].(string),
	}

	resp := map[string]interface{}{
		"data": pod,
		"code": http.StatusOK,
	}

	json.NewEncoder(w).Encode(resp)
}

func deleteSinglePod(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var podParam PodParam
	var namespace string
	json.Unmarshal(reqBody, &podParam)

	if len(podParam.Namespace) > 0 {
		namespace = "-n=" + podParam.Namespace
	}

	respPod, err := exec.Command("kubectl", "delete", "pods", podParam.PodName, namespace).Output()
	if err != nil {
		log.Println("Err: ", err)
		json.NewEncoder(w).Encode(err)
		return
	}
	strPod := string(respPod)

	resp := map[string]interface{}{
		"data": strPod,
		"code": http.StatusOK,
	}

	json.NewEncoder(w).Encode(resp)
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", homePage).Methods("GET")
	myRouter.HandleFunc("/pods", createNewPod).Methods("POST")
	myRouter.HandleFunc("/pods", returnAllPods).Methods("GET")
	myRouter.HandleFunc("/pods/{name}", returnSinglePod).Methods("GET")
	myRouter.HandleFunc("/pods/delete", deleteSinglePod).Methods("POST")
	log.Println("application running in :10000")
	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func main() {
	handleRequests()
}
