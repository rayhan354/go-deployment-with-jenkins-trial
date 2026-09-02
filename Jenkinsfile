pipeline {
    agent any

    environment {
        // ----- Customise these for your registry -----
        REGISTRY_URL   = 'docker.io/rayhan354'    // <-- change to your registry
        IMAGE_NAME     = 'myapp'
        // Credentials ID for registry login (stored in Jenkins)
        REGISTRY_CREDS = 'docker-credentials'     // <-- create this in Jenkins
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
                    def commitHash = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    sh """
                        docker build --build-arg VERSION=${commitHash} -t ${IMAGE_NAME}:${commitHash} .
                        docker tag ${IMAGE_NAME}:${commitHash} ${REGISTRY_URL}/${IMAGE_NAME}:${commitHash}
                    """
                }
            }
        }

        stage('Push (Simulated)') {
            steps {
                script {
                    def commitHash = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    // In a real pipeline, uncomment this block and set up credentials:
                    // withCredentials([usernamePassword(credentialsId: REGISTRY_CREDS, usernameVariable: 'DOCKER_USER', passwordVariable: 'DOCKER_PASS')]) {
                    //     sh """
                    //         docker login ${REGISTRY_URL} -u ${DOCKER_USER} -p ${DOCKER_PASS}
                    //         docker push ${REGISTRY_URL}/${IMAGE_NAME}:${commitHash}
                    //     """
                    // }
                    echo "✅ Push stage: would push ${REGISTRY_URL}/${IMAGE_NAME}:${commitHash}"
                    echo "🔑 Jenkins credential '${REGISTRY_CREDS}' (username/password) would be used."
                }
            }
        }

        stage('Deploy') {
            steps {
                script {
                    def commitHash = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()

                    // 1. Extract binary from the newly built image
                    sh """
                        docker create --name extract-${commitHash} ${IMAGE_NAME}:${commitHash}
                        docker cp extract-${commitHash}:/app /tmp/app-bin/app-new
                        docker rm extract-${commitHash}
                    """

                    // 2. Backup current binary (safe rollback)
                    sh """
                        if [ -f /tmp/app-bin/app ]; then
                            cp /tmp/app-bin/app /tmp/app-bin/app.bak
                        fi
                    """

                    // 3. Swap with the new binary
                    sh """
                        mv /tmp/app-bin/app-new /tmp/app-bin/app
                    """

                    // 4. Restart the container with the new binary mounted
                    sh """
                        # Stop and remove old container (if exists)
                        docker stop myapp || true
                        docker rm myapp || true

                        # Run a fresh container with volume mount
                        docker run -d \
                            --restart=unless-stopped \
                            -p 8080:8080 \
                            -v /tmp/app-bin/app:/app \
                            --name myapp \
                            ${IMAGE_NAME}:${commitHash}
                    """

                    // 5. Quick health check – ensure the container is running
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
                // Rollback if the deploy stage or health check fails
                echo "🚨 Deploy failed – initiating rollback..."
                sh """
                    # Restore the previous binary (if backup exists)
                    if [ -f /tmp/app-bin/app.bak ]; then
                        mv /tmp/app-bin/app.bak /tmp/app-bin/app
                    fi

                    # Restart container with the restored binary
                    docker stop myapp || true
                    docker rm myapp || true

                    # Use the *same* image tag (the binary is now rolled back)
                    docker run -d \
                        --restart=unless-stopped \
                        -p 8080:8080 \
                        -v /tmp/app-bin/app:/app \
                        --name myapp \
                        ${REGISTRY_URL}/${IMAGE_NAME}:${commitHash} || {
                            echo "⚠️ Fallback: using local image"
                            docker run -d --restart=unless-stopped -p 8080:8080 -v /tmp/app-bin/app:/app --name myapp ${IMAGE_NAME}:${commitHash}
                        }
                """
                echo "✅ Rollback completed. Container is running the previous version."
            }
        }
        success {
            echo "🎉 Pipeline succeeded! New version deployed."
        }
    }
}
