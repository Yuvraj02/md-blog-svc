// Pipeline: blog-service
// Script Path: backend/services/blog-service/Jenkinsfile
// Flow: build/push image -> k8s Atlas Job -> bump md-helm-values image.tag (Argo CD deploys)
//
// Jenkins credentials:
//   kubeconfig-prod          (secret file)  — cluster kubeconfig
//   github-helm-values       (username + password/token) — push to md-helm-values
// Cluster must already have Secret blog-service-db-url (key: url) in ns marketing-digest.

pipeline {
  agent any

  environment {
    SERVICE            = 'blog-service'
    ECR_NAME           = 'blog'
    HELM_VALUES_PATH   = 'blogs/prod/values.yaml'
    MIGRATE_JOB_TMPL   = 'backend/services/blog-service/deployments/ci-migrate-job.yaml'
    HELM_VALUES_REPO   = 'https://github.com/Yuvraj02/md-helm-values.git'
    K8S_NAMESPACE      = 'marketing-digest'
    AWS_REGION         = "${env.AWS_DEFAULT_REGION ?: 'ap-south-1'}"
    GO_IMAGE           = 'golang:1.25-alpine'
    BUF_IMAGE          = 'bufbuild/buf:1.47.2'
    SERVICE_DIR        = 'backend/services/blog-service'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
        script {
          // GIT_COMMIT is only reliable after checkout scm
          env.IMAGE_TAG = env.GIT_COMMIT.take(8)
          echo "IMAGE_TAG=${env.IMAGE_TAG}"
        }
      }
    }

    stage('Resolve ECR') {
      steps {
        script {
          // Prefer process env (docker -e MD_ACCOUNT_ID). Pipeline env.* often misses it.
          def account = System.getenv('MD_ACCOUNT_ID')?.trim()
          if (!account) {
            // EC2 IMDSv2 — no docker / no hardcoded account
            account = sh(
              script: '''
                set -euo pipefail
                TOKEN=$(curl -sS -f -X PUT "http://169.254.169.254/latest/api/token" \
                  -H "X-aws-ec2-metadata-token-ttl-seconds: 60")
                curl -sS -f -H "X-aws-ec2-metadata-token: ${TOKEN}" \
                  http://169.254.169.254/latest/dynamic/instance-identity/document \
                  | sed -n 's/.*"accountId"[[:space:]]*:[[:space:]]*"\\([^"]*\\)".*/\\1/p'
              ''',
              returnStdout: true
            ).trim()
          }
          if (!account) {
            error('Could not resolve AWS account id (MD_ACCOUNT_ID unset and IMDS failed)')
          }
          env.AWS_ACCOUNT_ID = account
          env.IMAGE_REPO = "${env.AWS_ACCOUNT_ID}.dkr.ecr.${env.AWS_REGION}.amazonaws.com/marketing-digest/${env.ECR_NAME}"
          echo "IMAGE=${env.IMAGE_REPO}:${env.IMAGE_TAG}"
        }
      }
    }

    stage('Deps') {
      steps {
        sh '''
          docker run --rm \
            -v "$PWD":/workspace -w "/workspace/${SERVICE_DIR}" \
            ${GO_IMAGE} \
            sh -c "apk add --no-cache git make && go version && make tidy"
        '''
      }
    }

    stage('Lint') {
      steps {
        sh '''
          docker run --rm \
            -v "$PWD":/workspace -w "/workspace/${SERVICE_DIR}" \
            ${GO_IMAGE} \
            sh -c "apk add --no-cache git make && make lint"
          docker run --rm \
            -v "$PWD/md-protos":/workspace -w /workspace \
            ${BUF_IMAGE} lint
        '''
      }
    }

    stage('Unit Tests') {
      steps {
        sh '''
          docker run --rm \
            -v "$PWD":/workspace -w "/workspace/${SERVICE_DIR}" \
            ${GO_IMAGE} \
            sh -c "apk add --no-cache git make && make test"
        '''
      }
    }

    stage('Docker Build') {
      steps {
        sh '''
          docker build \
            -f backend/services/blog-service/Dockerfile \
            -t ${IMAGE_REPO}:${IMAGE_TAG} \
            .
        '''
      }
    }

    stage('Docker Push') {
      steps {
        sh '''
          PASS=$(docker run --rm \
            -e AWS_DEFAULT_REGION \
            amazon/aws-cli:2.15.30 \
            ecr get-login-password --region "${AWS_REGION}")
          echo "$PASS" | docker login --username AWS --password-stdin \
            "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
          docker push ${IMAGE_REPO}:${IMAGE_TAG}
        '''
      }
    }

    stage('Migrate (k8s Job)') {
      steps {
        withCredentials([file(credentialsId: 'kubeconfig-prod', variable: 'KUBECONFIG')]) {
          sh '''
            set -euo pipefail
            JOB_NAME="blog-migrate-${IMAGE_TAG}"
            IMAGE="${IMAGE_REPO}:${IMAGE_TAG}"

            kubectl -n "${K8S_NAMESPACE}" delete job "${JOB_NAME}" --ignore-not-found
            sed -e "s|__JOB_NAME__|${JOB_NAME}|g" -e "s|__IMAGE__|${IMAGE}|g" \
              "${MIGRATE_JOB_TMPL}" | kubectl apply -f -

            echo "Waiting for Job ${JOB_NAME} (image ${IMAGE}) ..."
            if ! kubectl -n "${K8S_NAMESPACE}" wait --for=condition=complete "job/${JOB_NAME}" --timeout=300s; then
              kubectl -n "${K8S_NAMESPACE}" logs "job/${JOB_NAME}" || true
              kubectl -n "${K8S_NAMESPACE}" describe "job/${JOB_NAME}" || true
              exit 1
            fi
            kubectl -n "${K8S_NAMESPACE}" logs "job/${JOB_NAME}" || true
          '''
        }
      }
    }

    stage('Bump helm-values') {
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'github-helm-values',
          usernameVariable: 'GIT_USER',
          passwordVariable: 'GIT_TOKEN'
        )]) {
          sh '''
            set -euo pipefail
            rm -rf helm-values-work
            git clone --depth 1 \
              "https://${GIT_USER}:${GIT_TOKEN}@github.com/Yuvraj02/md-helm-values.git" \
              helm-values-work
            cd helm-values-work

            test -f "${HELM_VALUES_PATH}"
            sed -i -E "s/^(  tag: ).*/\\1\\"${IMAGE_TAG}\\"/" "${HELM_VALUES_PATH}"
            sed -i -E "s|^(  repository: ).*|\\1${IMAGE_REPO}|" "${HELM_VALUES_PATH}"

            git config user.email "jenkins@marketing-digest.local"
            git config user.name "jenkins"
            git add "${HELM_VALUES_PATH}"
            if git diff --cached --quiet; then
              echo "No helm-values change (tag already ${IMAGE_TAG})"
            else
              git commit -m "chore(blogs): bump image.tag to ${IMAGE_TAG}"
              git push origin HEAD:main
            fi
          '''
        }
      }
    }
  }

  post {
    failure {
      echo 'blog-service failed. helm-values is only bumped after the migration Job succeeds, so Argo CD will not roll out a broken migration.'
    }
    success {
      echo "Pushed ${IMAGE_REPO}:${IMAGE_TAG} and bumped md-helm-values ${HELM_VALUES_PATH}. Argo CD will sync the Deployment."
    }
  }
}
