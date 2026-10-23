#!/usr/bin/env groovy
// vim: ft=groovy
library 'status-jenkins-lib@v1.10.2'

def result = ''

pipeline {
  agent { label "${params.AGENT_LABEL} && nix" }

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
          result = nix.flake("status-go-library")
        }
      }
    }

    stage('Nix Cache') {
      steps { script {
        def cache = utils.isReleaseBuild() ? 'public' : 'ci'
        catchError(
          buildResult: 'SUCCESS',
          stageResult: 'UNSTABLE',
          catchInterruptions: false,
          message: 'Nix cache push failed',
        ) {
          nix.pushToAttic(derivations: [result], cache: cache)
        }
      } }
    }
  }

  post {
    cleanup {
      cleanWs()
      dir(env.WORKSPACE_TMP) { deleteDir() }
    }
  }
}
