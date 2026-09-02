pipeline {
    agent any

    environment {
        REGISTRY_URL   = 'docker.io/rayhan354'    // change to your registry
        IMAGE_NAME     = 'myapp'
        REGISTRY_CREDS = 'docker-credentials'
        // commitHash will be set dynamically in the Build stage
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
                    // Set commitHash as an environment variable
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
                    sh """
                        docker create --name extract-${env.commitHash} ${env.IMAGE_NAME}:${env.commitHash}
                        docker cp extract-${env.commitHash}:/app /tmp/app-bin/app-new
                        docker rm extract-${env.commitHash}
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
