pipeline {
  agent {
    node {
      label 'amd64 && ubuntu-1804 && overlay2'
    }
  }

  stages {
    stage('Build') {
      steps {
        sh 'make build'
      }
      post {
        success {
          echo 'successfully built'
        }
      }
    }

    stage('QA') {
      parallel {
        stage ('Lint') {
          steps {
            echo 'Running lint...'
            sh 'make lint'
          }
        }
        stage('Unit tests') {
          steps {
            sh 'make test'
          }
        }
      }
    }
    stage('notification') {
        steps {
            slackSend tokenCredentialId: 'slack-token-ducp-feed', color: 'good', channel: '#stacks-api', message: """
$CHANGE_AUTHOR: <$CHANGE_URL|PR: $CHANGE_ID> $CHANGE_TITLE
SUCCESS - See <$BUILD_URL/console|the Jenkins console for job $BUILD_ID>
"""
        }
    }
  }

  post {
    failure {
      echo "build failure"
      slackSend tokenCredentialId: 'slack-token-ducp-feed', color: 'danger', channel: '#stacks-api', message: """
$CHANGE_AUTHOR: <$CHANGE_URL|PR: $CHANGE_ID> $CHANGE_TITLE
FAILURE - See <$BUILD_URL/console|the Jenkins console for job $BUILD_ID>
"""
    }
  }
}
