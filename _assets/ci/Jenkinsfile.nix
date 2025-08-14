#!/usr/bin/env groovy
// vim: ft=groovy
library 'status-jenkins-lib@v1.9.26'

pipeline {
  agent { label "${params.AGENT_LABEL} && nix-2.24" }

  parameters {
    string(
      name: 'AGENT_LABEL',
      description: 'Label for targetted CI slave host.',
      defaultValue: params.AGENT_LABEL ?: jenkins.getAgentLabelFromJob(),
    )
  }

  options {
    disableConcurrentBuilds()
    disableRestartFromStage()
    /* manage how many builds we keep */
    buildDiscarder(logRotator(
      numToKeepStr: '20',
      daysToKeepStr: '30',
    ))
  }

  stages {
    stage('Build library') {
      steps {
        script {
          nix.flake("status-go-library")
        }
      }
    }

    stage('Build Android library') {
      when { expression { env.NODE_LABELS?.contains('linux') } }
      steps {
        script {
          nix.flake("status-go-mobile-android")
        }
      }
    }
  }

  post {
    cleanup { cleanWs() }
  }
}
