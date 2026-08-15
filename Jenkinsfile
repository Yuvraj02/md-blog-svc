// Pipeline for md-blog-svc (service repo root = workspace).
// Flow: build/push image -> k8s Atlas Job -> bump md-helm-values image.tag (Argo CD)
//
// Credentials: github-helm-values (username/token)
// Cluster access: /var/jenkins_home/.kube/config on the Jenkins host (no Jenkins credential needed)
//
// Jenkins itself runs in Docker with docker.sock. Sibling containers must mount the
// named volume "jenkins_home" (not -v $PWD) so they see the workspace files.

pipeline {
  agent any

  environment {
    SERVICE             = 'blog-service'
    ECR_NAME            = 'blog'
    HELM_VALUES_PATH    = 'blogs/prod/values.yaml'
    HELM_VALUES_REPO    = 'https://github.com/Yuvraj02/md-helm-values.git'
    MIGRATE_JOB_TMPL    = 'deployments/ci-migrate-job.yaml'
    K8S_NAMESPACE       = 'marketing-digest'
    AWS_REGION          = "${env.AWS_DEFAULT_REGION ?: 'ap-south-1'}"
    GO_IMAGE            = 'golang:1.25-alpine'
    JENKINS_HOME_VOLUME = 'jenkins_home'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
        script {
          env.IMAGE_TAG = env.GIT_COMMIT.take(8)
          echo "IMAGE_TAG=${env.IMAGE_TAG}"
          if (!fileExists('go.mod') || !fileExists('Dockerfile')) {
            error('Expected md-blog-svc repo root (go.mod + Dockerfile). Check Jenkins SCM URL.')
          }
        }
      }
    }

    stage('Resolve ECR') {
      steps {
        script {
          def account = sh(
            script: '''
              set -euo pipefail
              if [ -n "${MD_ACCOUNT_ID:-}" ]; then
                printf '%s' "$MD_ACCOUNT_ID"
                exit 0
              fi
              TOKEN=$(curl -sS -f -X PUT "http://169.254.169.254/latest/api/token" \
                -H "X-aws-ec2-metadata-token-ttl-seconds: 60")
              curl -sS -f -H "X-aws-ec2-metadata-token: ${TOKEN}" \
                http://169.254.169.254/latest/dynamic/instance-identity/document \
                | sed -n 's/.*"accountId"[[:space:]]*:[[:space:]]*"\\([^"]*\\)".*/\\1/p'
            ''',
            returnStdout: true
          ).trim()
          if (!account) {
            error('Could not resolve AWS account id')
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
          set -eux
          test -f go.mod
          cat > .ci-run.sh <<'EOF'
#!/bin/sh
set -e
apk add --no-cache git
go version
go mod tidy
EOF
          chmod +x .ci-run.sh
          docker run --rm \
            -e GOWORK=off \
            -v "${JENKINS_HOME_VOLUME}:/var/jenkins_home" \
            -w "${PWD}" \
            --entrypoint /bin/sh \
            "${GO_IMAGE}" \
            "${PWD}/.ci-run.sh"
        '''
      }
    }

    stage('Lint') {
      steps {
        sh '''
          set -eux
          cat > .ci-run.sh <<'EOF'
#!/bin/sh
set -e
apk add --no-cache git
go vet ./...
EOF
          chmod +x .ci-run.sh
          docker run --rm \
            -e GOWORK=off \
            -v "${JENKINS_HOME_VOLUME}:/var/jenkins_home" \
            -w "${PWD}" \
            --entrypoint /bin/sh \
            "${GO_IMAGE}" \
            "${PWD}/.ci-run.sh"
        '''
      }
    }

    stage('Unit Tests') {
      steps {
        sh '''
          set -eux
          cat > .ci-run.sh <<'EOF'
#!/bin/sh
set -e
apk add --no-cache git
go test ./...
EOF
          chmod +x .ci-run.sh
          docker run --rm \
            -e GOWORK=off \
            -v "${JENKINS_HOME_VOLUME}:/var/jenkins_home" \
            -w "${PWD}" \
            --entrypoint /bin/sh \
            "${GO_IMAGE}" \
            "${PWD}/.ci-run.sh"
        '''
      }
    }

    stage('Docker Build') {
      steps {
        sh 'DOCKER_BUILDKIT=0 docker build -t ${IMAGE_REPO}:${IMAGE_TAG} .'
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
        sh '''
          set -euo pipefail
          export KUBECONFIG=/var/jenkins_home/.kube/config
          test -f "$KUBECONFIG"
          command -v kubectl >/dev/null

          JOB_NAME="blog-migrate-${IMAGE_TAG}"
          IMAGE="${IMAGE_REPO}:${IMAGE_TAG}"

          kubectl -n "${K8S_NAMESPACE}" delete job "${JOB_NAME}" --ignore-not-found
          sed -e "s|__JOB_NAME__|${JOB_NAME}|g" -e "s|__IMAGE__|${IMAGE}|g" \
            "${MIGRATE_JOB_TMPL}" | kubectl apply -f -

          if ! kubectl -n "${K8S_NAMESPACE}" wait --for=condition=complete "job/${JOB_NAME}" --timeout=300s; then
            kubectl -n "${K8S_NAMESPACE}" logs "job/${JOB_NAME}" || true
            kubectl -n "${K8S_NAMESPACE}" describe "job/${JOB_NAME}" || true
            exit 1
          fi
          kubectl -n "${K8S_NAMESPACE}" logs "job/${JOB_NAME}" || true
        '''
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
              echo "No helm-values change"
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
      echo 'blog-service failed before or during migrate; helm-values was not bumped.'
    }
    success {
      echo "Pushed ${IMAGE_REPO}:${IMAGE_TAG}; Argo CD will sync from helm-values."
    }
  }
}
