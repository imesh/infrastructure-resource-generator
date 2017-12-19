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
	"fmt"
	"strconv"
)

const pathSeparator = string(os.PathSeparator)

type Port struct {
	Name            string
	Protocol        string
	Port            int
	HostPort        int
	External        bool
	SessionAffinity bool
}

type Ingress struct {
	Name  string
	Ports [] string
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
	Volumes []string
	Ports   [] Port
	Services [] struct {
		Name  string
		Ports [] string
	}
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
    		if port.Name == ingressPort {
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

func getDeployment(filePath string) *Deployment {
	log.Infoln("Reading deployment:", filePath)
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v %v", filePath, err)
	}
	var c *Deployment = new(Deployment)
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error parsing yaml file: %v %v", filePath, err)
	}
	return c
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

func usage() {
	fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n", )
	flag.PrintDefaults()
	os.Exit(2)
}

func init() {
	// Initialize glog
	flag.Usage = usage
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

}

func generate(executionPath string, deploymentsFolderPath string, filePath string) {
	deployment := getDeployment(filePath)
	componentNamesMap := map[string]bool{}

	// Generate dockerfiles
	for _, component := range deployment.Components {
		// Read component code name
		codeName := component.CodeName
		if codeName == "" {
			codeName = component.Name
		}

		if _, ok := componentNamesMap[codeName]; ok {
			// Dockerfile already generated for component
			continue
		}
		if component.Image != "" {
			// Docker image specified, do not require to generate dockerfile
			continue
		}

		componentNamesMap[codeName] = true
		templatePath := executionPath + pathSeparator + "templates" + pathSeparator + "docker" + pathSeparator + "Dockerfile.tmpl"
		outputFilePath := executionPath + pathSeparator + "output" + pathSeparator + "docker" + pathSeparator + codeName + pathSeparator + "Dockerfile"
		applyTemplate(templatePath, outputFilePath, component)
	}

	// Generate docker compose template
	templatePath := executionPath + pathSeparator + "templates" + pathSeparator + "docker-compose" + pathSeparator + "docker-compose.tmpl"
	outputFilePath := executionPath + pathSeparator + "output" + pathSeparator + "docker-compose"
	// Append sub folder path
	fileFolderPath := strings.Replace(filePath, filepath.Base(filePath), "", 1)
	subFolderPath := strings.Replace(fileFolderPath, deploymentsFolderPath, "", 1)
	if subFolderPath != "" {
		outputFilePath = outputFilePath + subFolderPath + "docker-compose.yml"
	} else {
		outputFilePath = outputFilePath + pathSeparator + "docker-compose.yml"
	}
	applyTemplate(templatePath, outputFilePath, deployment)
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
