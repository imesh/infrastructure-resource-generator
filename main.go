package main

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
	log "github.com/golang/glog"
	"os"
	"text/template"
	"strings"
	"path/filepath"
	"flag"
	"strconv"
)

const pathSeparator = string(os.PathSeparator)

type Port struct {
	Name            string
	Protocol        string
	Port            int
	HostPort        int
	External        bool
}

type Service struct {
	Component Component
	Name      string
	PortNames [] string
	Ports     [] Port
	SessionAffinity bool
}

type Ingress struct {
	Name      string
	PortNames [] string
	Ports     [] Port
}

type Component struct {
	Name         string
	CodeName     string
	Version      string
	Labels       [] map[string]interface{}
	Cpu          string
	Memory       string
	Disk         string
	Distribution string
	Entrypoint   string
	Image        string
	Replicas     int
	Scalable     bool
	Clustering   bool
	Environment  [] map[string]interface{}
	Databases [] struct {
		Name         string
		Type         string
		Version      string
		CreateScript string
	}
	Volumes [] struct {
		Name       string
		Type       string
		SourcePath string
		MountPath  string
	}
	Ports     [] Port
	Services  [] Service
	Ingresses [] Ingress
	Dependencies [] struct {
		Component string
		Ports     [] string
	}
	Healthcheck struct {
		Command struct {
			Assertion []string
		}
		TcpSocket struct {
			Port int
		}
		HttpGet struct {
			Path string
			Port int
		}
		InitialDelaySeconds int
		PeriodSeconds       int
		TimeoutSeconds      int
	}
}

type Deployment struct {
	ApiVersion string
	Kind       string
	Name       string
	Version    string
	Labels     [] map[string]interface{}
	Components [] Component
}

func (ingress Ingress) FindExcludePorts(ports [] Port) string {
	var excludePorts string
	for _, port := range ports {
		found := false
		for _, ingressPort := range ingress.Ports {
			if port.Name == ingressPort.Name {
				found = true
				break
			}
		}
		if !found {
			if excludePorts != "" {
				excludePorts = excludePorts + ","
			}
			excludePorts = excludePorts + strconv.Itoa(port.Port)
		}
	}
	return excludePorts
}

func (deployment Deployment) FindIngresses() []string {
	var ingresses [] string
	for _, component := range deployment.Components {
		if len(component.Ingresses) > 0 {
			ingresses = append(ingresses, component.Name)
		}
	}
	return ingresses
}

func (component Component) FindImage() string {
	if component.Image != "" {
		return component.Image
	}
	return component.CodeName + ":" + component.Version
}

func (component Component) FindPattern() string {
	// TODO: Find pattern
	return "pattern-1"
}

func (component Component) FindKubernetesImage() string {
	return component.CodeName + "-kubernetes:" + component.Version
}

func getDeployment(filePath string) Deployment {
	log.Infoln("Reading deployment:", filePath)
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v %v", filePath, err)
	}
	var deployment *Deployment = new(Deployment)
	err = yaml.Unmarshal(yamlFile, deployment)
	if err != nil {
		log.Fatalf("Error parsing yaml file: %v %v", filePath, err)
	}

	// Set service -> component references, service.ports
	for _, component := range deployment.Components {
		for _, service := range component.Services {
			service.Component = component
			service.Ports = [] Port{}
			for _, portName := range service.PortNames {
				port := findPort(component, portName)
				service.Ports = append(service.Ports, port)
			}
		}
	}
	return *deployment
}

func findPort(component Component, portName string) Port {
	for _, port := range component.Ports {
		if port.Name == portName {
			return port
		}
	}
	return Port{}
}

func applyTemplate(templateFilePath string, outputFilePath string, data interface{}) {
	log.V(2).Infoln("Applying template", templateFilePath)
	template, err := template.ParseFiles(templateFilePath)
	if err != nil {
		log.Error(err)
		return
	}

	lastIndex := strings.LastIndex(outputFilePath, string(os.PathSeparator))
	outputFolderPath := outputFilePath[0: lastIndex]
	os.MkdirAll(outputFolderPath, os.ModePerm);
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Error("Error creating file:", outputFilePath, err)
		return
	}

	log.Infoln("Creating file:", outputFilePath)
	err = template.Execute(outputFile, data)
	if err != nil {
		log.Error("Error executing template:", err)
		return
	}
	outputFile.Close()
}

func init() {
	// Initialize glog
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

}

// Generate infrastructure resources for a given deployment
func generate(executionPath string, deploymentsFolderPath string, deploymentFilePath string) {
	deployment := getDeployment(deploymentFilePath)
	deploymentFileFolderPath := strings.Replace(deploymentFilePath, filepath.Base(deploymentFilePath), "", 1)
	subFolderPath := strings.Replace(deploymentFileFolderPath, deploymentsFolderPath, "", 1)

	templatesFolderPath := executionPath + pathSeparator + "templates"
	outputFolderPath := executionPath + pathSeparator + "output"

	dockerFileTemplPath := templatesFolderPath + pathSeparator + "docker" + pathSeparator + "Dockerfile-template"
	dockerFileOutputFolderPath := outputFolderPath + pathSeparator + "docker"

	generateDockerFiles(dockerFileTemplPath, dockerFileOutputFolderPath, deployment)
	generateDockerComposeTemplate(templatesFolderPath, outputFolderPath, subFolderPath, deployment)
	generateKubernetesResources(templatesFolderPath, outputFolderPath, subFolderPath, deployment)
}

// Generate dockerfiles for components found in a given deployment
func generateDockerFiles(templateFilePath string, outputFolderPath string, deployment Deployment) {
	componentNamesMap := map[string]bool{}
	for _, component := range deployment.Components {
		// Read component code name
		codeName := component.CodeName
		if codeName == "" {
			codeName = component.Name
		}

		if _, ok := componentNamesMap[codeName]; ok {
			// Dockerfile already generated for the selected component
			continue
		}
		if component.Image != "" {
			// Docker image specified, do not need to generate a dockerfile
			continue
		}

		componentNamesMap[codeName] = true
		outputFilePath := outputFolderPath + pathSeparator + codeName + pathSeparator + "Dockerfile"
		applyTemplate(templateFilePath, outputFilePath, component)
	}
}

// Generate docker compose template for a given deployment
func generateDockerComposeTemplate(templateFolderPath string, outputFolderPath string, subFolderPath string, deployment Deployment) {
	templateFilePath := templateFolderPath + pathSeparator + "docker-compose" + pathSeparator + "docker-compose-template.yaml"
	outputFilePath := outputFolderPath + pathSeparator + "docker-compose"

	// Append sub folder path
	if subFolderPath != "" {
		outputFilePath = outputFilePath + subFolderPath + "docker-compose.yml"
	} else {
		outputFilePath = outputFilePath + pathSeparator + "docker-compose.yml"
	}
	applyTemplate(templateFilePath, outputFilePath, deployment)
}

func generateKubernetesResources(templateFolderPath string, outputFolderPath string, subFolderPath string, deployment Deployment) {
	k8sTemplateFolderPath := templateFolderPath + pathSeparator + "kubernetes"
	k8sOutputFolderPath := outputFolderPath + pathSeparator + "kubernetes" + subFolderPath

	k8sDockerFileTemplPath := k8sTemplateFolderPath + pathSeparator + "Dockerfile-template"
	k8sDockerFilesFolderPath := k8sOutputFolderPath + pathSeparator + "dockerfiles"

	generateDockerFiles(k8sDockerFileTemplPath, k8sDockerFilesFolderPath, deployment)
	for _, component := range deployment.Components {
		generateKubernetesDeployment(k8sTemplateFolderPath, k8sOutputFolderPath, component)
	}
}

func generateKubernetesDeployment(k8sTemplateFolderPath string, k8sOutputFolderPath string, component Component) {
	k8sDeploymentTemplPath := k8sTemplateFolderPath + pathSeparator + "deployment-template.yaml"
	k8sDeploymentOutputPath := k8sOutputFolderPath + component.Name + "-deployment.yaml"
	applyTemplate(k8sDeploymentTemplPath, k8sDeploymentOutputPath, component)

	for _, service := range component.Services {
		generateKubernetesService(k8sTemplateFolderPath, k8sOutputFolderPath, service)
	}
}

func generateKubernetesService(k8sTemplateFolderPath string, k8sOutputFolderPath string, service Service) {
	k8sServiceTemplPath := k8sTemplateFolderPath + pathSeparator + "service-template.yaml"
	k8sServiceOutputPath := k8sOutputFolderPath + service.Name + "-service.yaml"
	applyTemplate(k8sServiceTemplPath, k8sServiceOutputPath, service)
}

func main() {
	executionPath := os.Getenv("IRG_HOME");
	if (len(executionPath) <= 0) {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		executionPath = filepath.Dir(ex)
	}

	deploymentFolderPath := executionPath + "/examples"
	log.Infoln("Execution path: ", executionPath)
	log.Infoln("Deployments path: ", deploymentFolderPath)

	err := filepath.Walk(deploymentFolderPath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			generate(executionPath, deploymentFolderPath, path)
		}
		return nil
	})
	if err != nil {
		log.Error(err)
	}
}
