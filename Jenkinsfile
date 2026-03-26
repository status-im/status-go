#!/usr/bin/env groovy
library 'status-jenkins-lib@v1.9.16'

pipeline {
  agent {
    docker {
      label 'linuxcontainer'
      image 'harbor.status.im/infra/ci-build-containers:linux-base-1.0.0'
      args '--volume=/var/run/docker.sock:/var/run/docker.sock ' +
           '--user jenkins'
    }
  }

  options {
    disableConcurrentBuilds()
    /* manage how many builds we keep */
    buildDiscarder(logRotator(
      numToKeepStr: '20',
      daysToKeepStr: '30',
    ))
  }
  parameters {
    string(
      name: 'DOCKER_CRED',
      description: 'Name of Docker Registry credential.',
      defaultValue: params.DOCKER_CRED ?: 'harbor-status-im-robot',
    )
    string(
      name: 'DOCKER_REGISTRY_URL',
      description: 'URL of the Docker Registry',
      defaultValue: params.DOCKER_REGISTRY_URL ?: 'https://harbor.status.im',
    )
    string(
      name: 'IMAGE_TAG',
      description: 'Image tag',
      defaultValue: params.IMAGE_TAG ?: 'develop',
    )
    string(
      name: 'IMAGE_NAME',
      description: 'Name of the Docker image',
      defaultValue: 'status-im/status-backend',
    )
  }

  stages {
    stage('Bulding docker images') {
      steps {
        script {
          image = docker.build(
            "${params.IMAGE_NAME}:${params.IMAGE_TAG}", "./"
          )
        }
      }
    }
    stage('Push docker image'){
      steps {
        script {
          withDockerRegistry([
            credentialsId: params.DOCKER_CRED, url: params.DOCKER_REGISTRY_URL
          ]) {
            image.push()
          }
        }
      }
    }
  }

  post {
    cleanup { cleanWs() }
  }
}
