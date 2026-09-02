pipeline {
    agent any

    environment {
        REGISTRY_URL   = 'docker.io/rayhan354'
        IMAGE_NAME     = 'myapp'
        REGISTRY_CREDS = 'docker-credentials'
    }

    // Global variable to hold the commit hash
    def commitHash = ''

    stages {
        stage('Test') {
            steps {
                sh 'go test ./...'
            }
        }

        stage('Build') {
            steps {
                script {
                    // Assign commitHash globally
                    commitHash = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
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
                    // (Simulated push – uses commitHash from global scope)
                    echo "✅ Push stage: would push ${REGISTRY_URL}/${IMAGE_NAME}:${commitHash}"
                    echo "🔑 Jenkins credential '${REGISTRY_CREDS}' (username/password) would be used."
                }
            }
        }

        stage('Deploy') {
            steps {
                script {
                    // commitHash is now accessible
                    sh """
                        docker create --name extract-${commitHash} ${IMAGE_NAME}:${commitHash}
                        docker cp extract-${commitHash}:/app /tmp/app-bin/app-new
                        docker rm extract-${commitHash}
                    """
                    sh """
                        if [ -f /tmp/app-bin/app ]; then
                            cp /tmp/app-bin/app /tmp/app-bin/app.bak
                        fi
                    """
                    sh """
                        mv /tmp/app-bin/app-new /tmp/app-bin/app
                    """
                    sh """
                        docker stop myapp || true
                        docker rm myapp || true
                        docker run -d \
                            --restart=unless-stopped \
                            -p 8080:8080 \
                            -v /tmp/app-bin/app:/app \
                            --name myapp \
                            ${IMAGE_NAME}:${commitHash}
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
                // commitHash is available here as well
                echo "🚨 Deploy failed – initiating rollback..."
                sh """
                    if [ -f /tmp/app-bin/app.bak ]; then
                        mv /tmp/app-bin/app.bak /tmp/app-bin/app
                    fi
                    docker stop myapp || true
                    docker rm myapp || true
                    docker run -d \
                        --restart=unless-stopped \
                        -p 8080:8080 \
                        -v /tmp/app-bin/app:/app \
                        --name myapp \
                        ${IMAGE_NAME}:${commitHash}
                """
                echo "✅ Rollback completed."
            }
        }
        success {
            echo "⭕️ Pipeline succeeded! New version deployed."
        }
    }
}
