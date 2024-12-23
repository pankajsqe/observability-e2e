package utils

import (
	"log"
	"os"

	rancher "github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/kubectl"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

func DeployPrometheusRule(mySession *rancher.Client, yamlPath string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	prometheusRuleApply, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully fetchall: %v", prometheusRuleApply)

	return nil
}

func DeployAlertManagerConfig(mySession *rancher.Client, yamlPath string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	alertManagerConfigApply, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully fetchall: %v", alertManagerConfigApply)

	return nil
}
