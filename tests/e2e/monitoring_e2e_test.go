package e2e_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/observability-e2e/tests/helper/utils"
	rancher "github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/kubectl"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// Define the struct for the Alert
type Alert struct {
	Annotations  map[string]string `json:"annotations"`
	EndsAt       string            `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	Receivers    []Receiver        `json:"receivers"`
	StartsAt     string            `json:"startsAt"`
	Status       AlertStatus       `json:"status"`
	UpdatedAt    string            `json:"updatedAt,omitempty"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	Labels       map[string]string `json:"labels"`
}

// Define the struct for Receiver
type Receiver struct {
	Name string `json:"name"`
}

// Define the struct for AlertStatus
type AlertStatus struct {
	InhibitedBy []string `json:"inhibitedBy"`
	SilencedBy  []string `json:"silencedBy"`
	State       string   `json:"state"`
}

const (
	defaultRandStringLength  = 5
	prometheusRulesSteveType = "monitoring.coreos.com.prometheusrule"
	prometheusRuleFilePath   = "../helper/yamls/createPrometheusRule.yaml"
)

// var ruleLabel = map[string]string{"team": "qa"}

var _ = Describe("Observability Monitoring E2E Test Suite", func() {
	var clientWithSession *rancher.Client //RancherConfig *Config

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Test : Verify Creating prometheus rule using kubectl", Label("LEVEL1", "monitoring", "E2E", "PromFed"), func() {

		By("1) Apply yaml to create prometheus rule")
		prometheusError := utils.DeployPrometheusRule(clientWithSession, prometheusRuleFilePath)

		if prometheusError != nil {
			e2e.Logf("Failed to deploy Prometheus rule: %v", prometheusError)
		} else {
			e2e.Logf("Prometheus Rule deployed successfully!")
		}

		By("2) Fetch all the prometheus rule")
		fetchPrometheusRules := []string{"kubectl", "get", "prometheusRule", "test-prometheus-rule", "-n", "cattle-monitoring-system"}
		verifyPetchPrometheusRules, err := kubectl.Command(clientWithSession, nil, "local", fetchPrometheusRules, "")
		if err != nil {
			e2e.Failf("Failed to fetch PrometheusRule 'test-prometheus-rule'. Error: %v", err)
		}
		e2e.Logf("Successfully fetched PrometheusRule: %v", verifyPetchPrometheusRules)
	})

	It("Test : Verify default Watchdog alert is present", Label("LEVEL1", "monitoring", "E2E"), func() {

		By("1) Create a container to access curl")
		creatContainer := []string{"kubectl", "run", "test", "--image=ranchertest/mytestcontainer", "-n", "default"}
		_, err := kubectl.Command(clientWithSession, nil, "local", creatContainer, "")
		if err != nil {
			e2e.Failf("Failed to create container. Error: %v", err)
		}

		time.Sleep(30 * time.Second)

		By("2) Fetching alerts via Curl request")
		curl := []string{"kubectl", "exec", "test", "-n", "default", "--", "curl", "-s", "http://rancher-monitoring-alertmanager.cattle-monitoring-system:9093/api/v2/alerts"}
		output, err := kubectl.Command(clientWithSession, nil, "local", curl, "")
		output = strings.TrimSpace(output)
		if err != nil {
			e2e.Failf("Failed to get curl response. Error: %v", err)
		}

		By("3) Unmarshalling json output response")
		var alerts []Alert
		if err := json.Unmarshal([]byte(output), &alerts); err != nil {
			e2e.Failf("Failed to unmarshal JSON response. Error: %v", err)
		}

		By("4) Search for the Watchdog alert")
		var watchdogAlert *Alert
		for _, alert := range alerts {
			if alert.Labels["alertname"] == "Watchdog" {
				watchdogAlert = &alert
				break
			}
		}

		By("5)Assert if the Watchdog alert was found ")
		if watchdogAlert == nil {
			e2e.Failf("Expected 'Watchdog' alert not found in response")
		}

		defer func() {
			By("6) Deleting the container")
			deleteContainer := []string{"kubectl", "delete", "pod", "test", "-n", "default"}
			deleteConfirm, err := kubectl.Command(clientWithSession, nil, "local", deleteContainer, "")
			if err != nil {
				e2e.Logf("Failed to delete container. Error: %v", err)
			} else {
				e2e.Logf("Verified container is deleted %v", deleteConfirm)
			}
		}()

	})

	It("Test : Verify status of rancher-monitoring pods using kubectl", Label("LEVEL1", "monitoring", "E2E"), func() {

		By("0) Fetch all the pods belongs to rancher-monitoring")
		fetchPods := []string{"kubectl", "get", "pods", "-n", "cattle-monitoring-system", "--no-headers"}
		rancherMonitoringPods, err := kubectl.Command(clientWithSession, nil, "local", fetchPods, "")
		if err != nil {
			e2e.Failf("Failed to get pods . Error: %v", err)
		}

		By("1) Read all the pods and verify the status of rancher-monitoring-Pods")
		pods := strings.Split(rancherMonitoringPods, "\n")
		for _, pod := range pods {
			if pod == "" {
				continue
			}

			fields := strings.Fields(pod) // Split the line into pod name and its current status
			if len(fields) < 3 {
				e2e.Failf("Unexpected output format for pod: %s", pod)
			}

			podName := fields[0]
			podStatus := fields[2]

			if (podStatus != "Running") && (!strings.HasPrefix(podName, "rancher-monitoring")) { // Check if pod status is not 'Running'
				e2e.Failf("Pod %s is not in 'Running' state, current state: %s", podName, podStatus)
			}
		}

	})

	It("Test : Verify status of rancher-monitoring Deployments using kubectl", Label("LEVEL1", "monitoring", "E2E"), func() {

		By("0) Fetch all the deployments belonging to rancher-monitoring")
		fetchDeployments := []string{"kubectl", "get", "deployments", "-n", "cattle-monitoring-system", "--no-headers"}
		rancherMonitoringDeployments, err := kubectl.Command(clientWithSession, nil, "local", fetchDeployments, "")
		if err != nil {
			e2e.Failf("Failed to get deployments. Error: %v", err)
		}

		By("1) Read all the deployments and verify the status of rancher-monitoring deployments")
		deployments := strings.Split(rancherMonitoringDeployments, "\n")
		for _, deployment := range deployments {
			if deployment == "" {
				continue
			}

			fields := strings.Fields(deployment) // Split the line into deployment name and its current status
			if len(fields) < 4 {
				e2e.Failf("Unexpected output format for deployment: %s", deployment)
			}

			deploymentName := fields[0]
			readyReplicas := fields[1]
			availableReplicas := fields[3]

			readyCount := strings.Split(readyReplicas, "/")[0]
			desiredCount := strings.Split(readyReplicas, "/")[1]

			if availableReplicas != desiredCount && !strings.HasPrefix(deploymentName, "rancher-monitoring") {
				e2e.Failf("Deployment %s is not fully available. Desired: %s, Available: %s", deploymentName, desiredCount, availableReplicas)
			}

			if readyCount != desiredCount && !strings.HasPrefix(deploymentName, "rancher-monitoring") {
				e2e.Failf("Deployment %s is not fully ready. Desired: %s, Ready: %s", deploymentName, desiredCount, readyCount)
			}
		}

	})

	It("Test : Verify status of rancher-monitoring DaemonSets using kubectl", Label("LEVEL1", "monitoring", "E2E"), func() {

		By("0) Fetch all the daemon sets belongs to rancher-monitoring")
		fetchPods := []string{"kubectl", "get", "daemonsets", "-n", "cattle-monitoring-system", "--no-headers"}
		rancherMonitoringDaemonSets, err := kubectl.Command(clientWithSession, nil, "local", fetchPods, "")
		if err != nil {
			e2e.Failf("Failed to get daemonsets . Error: %v", err)
		}

		By("1) Read all the daemonSet and verify the status of rancher-monitoring-daemonSets")
		daemonSets := strings.Split(rancherMonitoringDaemonSets, "\n")
		for _, daemonSet := range daemonSets {
			if daemonSet == "" {
				continue
			}

			fields := strings.Fields(daemonSet) // Split the line into pod name and its current status
			if len(fields) < 6 {
				e2e.Failf("Unexpected output format for daemonSet: %s", daemonSet)
			}

			daemonSetName := fields[0]
			desiredPods := fields[1]
			readyPods := fields[3]
			availablePods := fields[5]

			if desiredPods != availablePods {
				e2e.Failf("DaemonSet %s is not fully available. Desired: %s, Available: %s", daemonSetName, desiredPods, availablePods)
			}

			if readyPods != desiredPods {
				e2e.Failf("DaemonSet %s is not fully ready. Desired: %s, Ready: %s", daemonSetName, desiredPods, readyPods)
			}
		}

	})

	It("Test: Verify newly created Prometheus rule alert is present", Label("LEVEL1", "monitoring", "E2E", "PromFed"), func() {

		By("1) Creating a container for curl access")
		createContainerCommand := []string{"kubectl", "run", "curl-container", "--image=ranchertest/mytestcontainer", "-n", "default"}
		_, err := kubectl.Command(clientWithSession, nil, "local", createContainerCommand, "")
		if err != nil {
			e2e.Failf("Failed to create container. Error: %v", err)
		}

		// Wait for the container to be ready
		var prometheusRuleAlert *Alert
		var maxRetries = 4
		var retryInterval = 20 * time.Second
		var attempt = 0

		for attempt < maxRetries {
			time.Sleep(30 * time.Second)
			By("2) Fetching alerts using curl request")
			curlCommand := []string{"kubectl", "exec", "curl-container", "-n", "default", "--", "curl", "-s", "http://rancher-monitoring-alertmanager.cattle-monitoring-system:9093/api/v2/alerts"}
			curlResponse, err := kubectl.Command(clientWithSession, nil, "local", curlCommand, "")
			curlResponse = strings.TrimSpace(curlResponse)
			if err != nil {
				e2e.Failf("Failed to get curl response. Error: %v", err)
			}

			By("3) Unmarshalling JSON response")
			var alerts []Alert
			if err := json.Unmarshal([]byte(curlResponse), &alerts); err != nil {
				e2e.Failf("Failed to unmarshal JSON response. Error: %v", err)
			}

			alertNamePattern := regexp.MustCompile("test-qa")

			By("4) Searching for the newly created Prometheus rule alert")

			// Search for the alert in the current iteration
			for _, alert := range alerts {
				e2e.Logf("Checking alerts line wise ----> %v", alert)
				if alertNamePattern.MatchString(alert.Labels["alertname"]) {
					prometheusRuleAlert = &alert
					break
				}
			}

			if prometheusRuleAlert != nil {
				e2e.Logf("Prometheus alert %v found on attempt: %v", prometheusRuleAlert, attempt+1)
				break
			} else {
				e2e.Logf("Prometheus alert not found. Retrying... (Attempt %d/%d)\n", attempt+1, maxRetries)
				attempt++

				if attempt < maxRetries {
					time.Sleep(retryInterval)
				}
			}
		}

		By("5) Verifying if the Prometheus rule alert was found")
		if prometheusRuleAlert == nil {
			e2e.Failf("Expected Prometheus rule alert not found in the response")
		} else {
			e2e.Logf("Found Prometheus rule alert: %+v\n", prometheusRuleAlert)
		}

		defer func() {
			By("6) Deleting the test container")
			deleteContainerCommand := []string{"kubectl", "delete", "pod", "curl-container", "-n", "default"}
			deleteResponse, err := kubectl.Command(clientWithSession, nil, "local", deleteContainerCommand, "")
			if err != nil {
				e2e.Logf("Failed to delete container. Error: %v", err)
			} else {
				e2e.Logf("Verified container is deleted: %v", deleteResponse)
			}
		}()

	})

})
