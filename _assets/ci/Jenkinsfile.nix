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
    booleanParam(
      name: 'USE_NWAKU',
      description: 'Whether to build with nwaku or not',
      defaultValue: params.USE_NWAKU ?: getNWakuMode(),
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
    USE_NWAKU = "${params.USE_NWAKU}"
    /* nwaku source directory */
    NWAKU_SOURCE_DIR = "${WORKSPACE_TMP}/nwaku"
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

/* We extract the name of the job from currentThread because
 * before an agent is picked env is not available. */
def getJobPathTokens() {
  return Thread.currentThread().getName().split('/')
}

def getNWakuMode() {
  return getJobPathTokens().any { it.contains('nwaku') }
}
