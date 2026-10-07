# Go Deployment with Docker and Jenkins

## Project Overview

This repository contains a simple Go HTTP server built, containerized, and deployed using Docker. The project demonstrates:

- Multi-stage Docker builds with version injection
- Minimal container images using `scratch`
- Binary swapping without rebuilding images (hotfix pattern)
- CI/CD pipeline automation with Jenkins

---

## Part I: Multi-stage Dockerfile & Build

### Environment Setup

- **OS**: Arch Linux
- **Tools**: Git, Docker, Go, curl, Jenkins
- **Project Path**: `/mnt/sdd1/Tes PT Simple Journey Indonesia/shift-test/`

### Application Code (`main.go`)

```go
package main

import (
    "fmt"
    "net/http"
)

var version string

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Version: %s", version)
}

func main() {
    http.HandleFunc("/", handler)
    fmt.Println("Server is running on port 8080...")
    http.ListenAndServe(":8080", nil)
}
```

### Dockerfile (Multi-stage)

```dockerfile
FROM golang:1.21-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY main.go .
RUN go mod init app

ARG VERSION=dev
RUN go build -ldflags="-X main.version=${VERSION}" -o app .

FROM scratch
COPY --from=builder /app/app /app
EXPOSE 8080
CMD ["/app"]
```

**Why `scratch`?**
- Smallest possible image (only the binary).
- Minimal attack surface – no OS, no shell, no packages.
- Static binary has no runtime dependencies.

### Build Command

```bash
docker build --build-arg VERSION=1.0.0 -t myapp:1.0.0 .
```

**Image Size**:

```
myapp:1.0.0   6.72MB
```

The image is \~6.72 MB because it uses scratch as the final base image. scratch is an empty image (0 bytes). The only content inside is the statically compiled Go binary (/app). The binary itself is \~6.7 MB because it includes the Go runtime, garbage collector, scheduler, and HTTP/networking libraries — all compiled into a single executable.

---

## Part II: Deploy & Binary Swap

### Run Container with Restart Policy

```bash
docker run -d \
  --restart=unless-stopped \
  -p 8080:8080 \
  --name myapp \
  myapp:1.0.0
```

**Verify**:

```bash
$ curl http://localhost:8080
Version: 1.0.0
```

### Build New Binary (Version 2.0.0)

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=2.0.0" -o app .
mkdir -p /tmp/app-bin
cp app /tmp/app-bin/app
```

### Swap Binary Without Rebuilding

```bash
docker stop myapp
docker rm myapp

docker run -d \
  --restart=unless-stopped \
  -p 8080:8080 \
  -v /tmp/app-bin/app:/app \
  --name myapp \
  myapp:1.0.0
```

**Verify**:

```bash
$ curl http://localhost:8080
Version: 2.0.0
```

### Curl Outputs – Before and After

| State | Output |
| :--- | :--- |
| Before swap | `Version: 1.0.0` |
| After swap | `Version: 2.0.0` |

### Why This Approach Fits a Production Hotfix

- The volume‑mount approach allows swapping the binary by simply replacing a file on the host and restarting the container. This works even with a `scratch`‑based image, requires no extra tooling inside the container, and limits downtime to the restart duration (a few seconds). It also makes rollback trivial – just put back the old binary and restart again. This pattern is common in production when you need to apply a hot‑fix quickly without rebuilding and redeploying the entire image.

---

## Part III: Jenkins CI/CD Pipeline

### Pipeline Stages

| Stage | Description |
| :--- | :--- |
| **Test** | Runs `go test ./...`. Fails pipeline if tests fail. |
| **Build** | Builds Docker image with Git commit hash as version tag. |
| **Push (Simulated)** | Simulates pushing the image to a registry. Uses Jenkins credentials. |
| **Deploy** | Extracts binary from the built image, mounts it via volume, and restarts the container. |

### Jenkinsfile

```groovy
pipeline {
    agent any

    environment {
        REGISTRY_URL   = 'docker.io/rayhan354'
        IMAGE_NAME     = 'myapp'
        REGISTRY_CREDS = 'docker-credentials'
        BIN_DIR        = "${env.WORKSPACE}/app-bin"
    }

    stages {
        stage('Test') {
            steps {
                sh 'go test ./...'
            }
        }

        stage('Build') {
            steps {
                script {
                    env.commitHash = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    sh """
                        docker build --build-arg VERSION=${env.commitHash} -t ${env.IMAGE_NAME}:${env.commitHash} .
                        docker tag ${env.IMAGE_NAME}:${env.commitHash} ${env.REGISTRY_URL}/${env.IMAGE_NAME}:${env.commitHash}
                    """
                }
            }
        }

        stage('Push (Simulated)') {
            steps {
                script {
                    echo "✅ Push stage: would push ${env.REGISTRY_URL}/${env.IMAGE_NAME}:${env.commitHash}"
                    echo "🔑 Jenkins credential '${env.REGISTRY_CREDS}' (username/password) would be used."
                }
            }
        }

        stage('Deploy') {
            steps {
                script {
                    sh "mkdir -p ${env.BIN_DIR}"
                    sh """
                        docker create --name extract-${env.commitHash} ${env.IMAGE_NAME}:${env.commitHash}
                        docker cp extract-${env.commitHash}:/app ${env.BIN_DIR}/app-new
                        docker rm extract-${env.commitHash}
                    """
                    sh """
                        if [ -f ${env.BIN_DIR}/app ]; then
                            cp ${env.BIN_DIR}/app ${env.BIN_DIR}/app.bak
                        fi
                    """
                    sh "mv ${env.BIN_DIR}/app-new ${env.BIN_DIR}/app"
                    sh """
                        docker stop myapp || true
                        docker rm myapp || true
                        docker run -d \
                            --restart=unless-stopped \
                            -p 8080:8080 \
                            -v ${env.BIN_DIR}/app:/app \
                            --name myapp \
                            ${env.IMAGE_NAME}:${env.commitHash}
                    """
                    sh """
                        sleep 3
                        if ! docker ps | grep -q myapp; then
                            echo "❌ Container failed to start, triggering rollback..."
                            exit 1
                        fi
                    """
                }
            }
        }
    }

    post {
        failure {
            script {
                echo "🚨 Deploy failed – initiating rollback..."
                sh """
                    if [ -f ${env.BIN_DIR}/app.bak ]; then
                        mv ${env.BIN_DIR}/app.bak ${env.BIN_DIR}/app
                    fi
                    docker stop myapp || true
                    docker rm myapp || true
                    docker run -d \
                        --restart=unless-stopped \
                        -p 8080:8080 \
                        -v ${env.BIN_DIR}/app:/app \
                        --name myapp \
                        ${env.IMAGE_NAME}:${env.commitHash}
                """
                echo "✅ Rollback completed."
            }
        }
        success {
            echo "🎉 Pipeline succeeded! New version deployed."
        }
    }
}
```

### Successful Pipeline Run

A successful pipeline run was executed with all stages passing:

```
Started by user Rayhan
[Pipeline] stage (Test) - ✅ ok
[Pipeline] stage (Build) - ✅ Image built and tagged
[Pipeline] stage (Push (Simulated)) - ✅ Simulated push
[Pipeline] stage (Deploy) - ✅ Container deployed
[Pipeline] End of Pipeline
Finished: SUCCESS
```

**Console output**: See `console-output-success.txt` in the repository root.

### Rollback Strategy

**If the deploy stage fails mid‑way**, the pipeline automatically triggers a rollback:

- The previous binary is restored from the backup (`app.bak`).
- The container is stopped and removed.
- A new container is started with the restored binary mounted, using the same (newly built) image.
- This ensures zero configuration drift – the image stays unchanged, and the rollback is as simple as swapping a file and restarting.

**Why this works**: Because the binary is mounted from the host, the container runs the old version even though the image tag is new. This makes rollback fast and reliable.

### Jenkins Credentials

- **GitHub**: `test-jenkins` credential (username + Personal Access Token)
- **Docker Registry**: `docker-credentials` (username + password) – simulated in this test

**No hardcoded secrets** are present in the Jenkinsfile.

---

## Obstacles Overcome

| Issue | Solution |
| :--- | :--- |
| **`DockerFile` vs `Dockerfile`** – case sensitivity | Renamed to `Dockerfile`. |
| **Missing `go.mod`** – Go 1.21+ requires modules | Added `RUN go mod init app` in Dockerfile and committed `go.mod`. |
| **Port 8080 in use** – stale `docker-proxy` | The Go app and Jenkins both default to port 8080. Because the Go app was running first, Jenkins automatically moved to port 8090 (configured in /etc/conf.d/jenkins). Later, a stale docker-proxy process (leftover from a previous container) blocked the Go app from restarting on 8080. Stopping and removing the conflicting container freed the port, allowing the Go app to bind to 8080 again. |
| **Mullvad VPN blocking Docker traffic** | Used `mullvad lan set allow` or temporarily disabled kill‑switch. |
| **Docker permission denied** in Jenkins | Added `jenkins` user to `docker` group and restarted Jenkins. |
| **`docker cp` permission denied** to `/tmp/app-bin` | Switched to workspace directory (`${env.WORKSPACE}/app-bin`) for binary mounts. |

---

## Repository Structure

```
shift-test/
├── main.go
├── main_test.go
├── Dockerfile
├── Jenkinsfile
├── go.mod
├── console-output-success.txt
└── README.md
```

---

## Submission Checklist

- ✔️ All code, Dockerfile, and Jenkinsfile pushed to GitHub.
- ✔️ Collaborator `admin@simplejourney.co.id` invited to the repository.
- ✔️ README.md includes build, run, deploy instructions, and write-ups.
- ✔️ Screenshot/log of successful pipeline run included.
- ✔️ Rollback strategy documented.

---

**Repository URL**: https://github.com/rayhan354/technical-test-shift-engineer
