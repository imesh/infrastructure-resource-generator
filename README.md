# Infrastructure Resource Generator

Infrastructure resource generator is a command line tool for generating resources required for 
deploying WSO2 middleware on well known infrastructure platforms. It makes use of a deployment 
specification for defining the deployment architecture and a set of templates for generating 
resources required. Currently, it supports Docker and Docker Compose. It will be extended to
support Kubernetes, OpenShift, Pivotal Cloud Foundry, DC/OS, AWS, Azure and Google Cloud.

## Getting Started

Follow below steps to get started:

1. Install the [Go tools](https://golang.org/doc/install) by following the official installation guide.

2. Clone this repository:
   
   ```
   git clone https://github.com/wso2-incubator/infrastructure-resource-generator
   ```

3. Build the source code:

   ````bash
   cd infrastructure-resource-generator
   go build
   ````

4. Run the binary:

   ```bash
   ./infrastructure-resource-generator
   ```

5. Switch to the output folder and view the generated files:

   ```bash
   cd output/
   output$ tree
    .
    └── docker
        ├── wso2am
        │   └── Dockerfile
        └── wso2am-analytics
            └── Dockerfile
   ```

## Deployment Specification

The archigos deployment specification has been designed according to standards and guidelines used by Docker, Docker Compose, Kubernetes, DC/OS, AWS Cloud Formation to be able to provide a generic definition for any software deployment:

````yaml
specVersion: 0.1
kind: Deployment
name: Name of the deployment # string
version: Version of the deployment # string
components:  # List of components
- name: Name # string
  codeName: Code name # string 
  version: Version # string
  cpu: Number of CPUs required # integer
  memory: Amount of memory required  # string
  disk: Amount of disk space required # string
  distribution: Distribution file name # string
  entrypoint: Startup script # string
  image: VM or container image if third party component # string
  replicas: Number of replicas # integer
  scalable: Scalable or not # boolean
  clustering: Clustering needed or not # boolean
  databases:
  - name: Database name # string
    type: Database type; MySQL, Postgres, Oracle, MSMSQL, MariaDB # string
    version: Database version # string
    createScript: Path of the database creation script # string
  volumes:
  - source path:destination path # string
  ports:
  - name: Port name # string
    protocol: Protocol of the port # string
    port: Port number exposed by the server # integer
    external: Port needs to be exposed externally or not # boolean
    sessionAffinity: Session affinity required or not # boolean
  services:
  - name: Service name # string
    ports:
    - Port name reference # string
  ingresses:
  - name: Ingress name # string
    ports:
    - Port name reference # string
  dependencies:
  - component: Dependent component code name reference # string
    ports:
    - Name of the dependent component port used # string
  healthcheck:
    # Define either httpGet or tcpSocket
    httpGet:
      path: Context path of the HTTP endpoint # string
      port: Port of the HTTP endpoint # integer
    tcpSocket:
      port: TCP port to be used by the health check # integer
    initialDelaySeconds: Initial delay in seconds # integer
    periodSeconds: Period in seconds # integer
````
