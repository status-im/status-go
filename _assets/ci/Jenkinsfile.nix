#!/usr/bin/env groovy
// vim: ft=groovy
library 'status-jenkins-lib@v1.9.39'

pipeline {
  agent { label "${params.AGENT_LABEL} && nix-2.33" }

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
      numToKeepStr: '5',
      daysToKeepStr: '30',
      artifactNumToKeepStr: '1',
      artifactDaysToKeepStr: '30',
    ))
  }

  environment {
    /* fixes nim cache collision */
    XDG_CACHE_HOME = "${env.WORKSPACE_TMP}/.cache"
  }

  stages {
    stage('Build library') {
      steps {
        script {
          nix.flake("status-go-library")
        }
      }
    }
  }

  post {
    cleanup {
      cleanWs()
      dir(env.WORKSPACE_TMP) { deleteDir() }
    }
  }
}

