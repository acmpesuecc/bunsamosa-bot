pipeline {
    agent any

    tools {
        // Ensure Go is installed and configured in Jenkins -> Manage Jenkins -> Global Tool Configuration
        go '1.23.2'  // replace with the name you configured in Jenkins
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build') {
            steps {
                sh 'go mod tidy'
                sh 'go build ./...'
            }
        }
    }
}
