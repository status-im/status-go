#!/usr/bin/env groovy

@NonCPS
def getVersion(branch, sha) {
    return branch.replaceAll(/\//, '-') + '-' + sha
}

node {
    checkout scm

    def remoteOriginRegex = ~/^remotes\/origin\//

    gitSHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
    gitShortSHA = gitSHA.take(7)
    String gitBranch = sh(returnStdout: true, script: 'git name-rev --name-only HEAD').trim() - remoteOriginRegex

    stage('Debug') {
        sh 'env'
        println(gitBranch)
        println(gitSHA)
    }

    stage('Build') {
        sh 'make statusgo-android'
    }

    stage('Deploy') {
        env.version = getVersion(gitBranch, gitShortSHA)

        withCredentials([usernameColonPassword(credentialsId: 'artifactory-deploy-bot', variable: 'USERPASS')]) {
            sh '''
                set +x
                curl -u "${USERPASS}" \
                    -X PUT "http://139.162.11.12:8081/artifactory/libs-release-local/status-im/status-go/${version}/status-go-${version}.aar" \
                    -T ./build/bin/statusgo-android-16.aar
            '''
        }
    }
}
