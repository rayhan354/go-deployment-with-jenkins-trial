pipeline {
    agent any

    environment {
        REGISTRY_URL   = 'docker.io/rayhan354'
        IMAGE_NAME     = 'myapp'
        REGISTRY_CREDS = 'docker-credentials'
        BIN_DIR        = "${env.WORKSPACE}/app-bin"   // use workspace directory
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
                    // Ensure the binary directory exists
                    sh """
                        mkdir -p ${env.BIN_DIR}
                    """

                    // Extract the binary
                    sh """
                        docker create --name extract-${env.commitHash} ${env.IMAGE_NAME}:${env.commitHash}
                        docker cp extract-${env.commitHash}:/app ${env.BIN_DIR}/app-new
                        docker rm extract-${env.commitHash}
                    """

                    // Backup existing binary if present
                    sh """
                        if [ -f ${env.BIN_DIR}/app ]; then
                            cp ${env.BIN_DIR}/app ${env.BIN_DIR}/app.bak
                        fi
                    """

                    // Swap with the new binary
                    sh """
                        mv ${env.BIN_DIR}/app-new ${env.BIN_DIR}/app
                    """

                    // Restart container with the new binary mounted
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

                    // Health check
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
                    # Restore backup if available
                    if [ -f ${env.BIN_DIR}/app.bak ]; then
                        mv ${env.BIN_DIR}/app.bak ${env.BIN_DIR}/app
                    fi

                    # Restart container with the restored binary
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
