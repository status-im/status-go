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

    // TODO(adam): enable when unit tests start passing
    // stage('Test') {
    //     sh 'make ci'
    // }

    stage('Build') {
        sh 'make statusgo-android'
    }

    stage('Deploy') {
        def version = getVersion(gitBranch, gitShortSHA)
        def server = Artifactory.server 'artifactory'
        def uploadSpec = """{
            "files": [
                {
                    "pattern": "build/bin/statusgo-android-16.aar",
                    "target": "libs-release-local/status-im/status-go/${version}/status-go-${version}.aar"
                }
            ]
        }"""

        // TODO(adam): do we need a POM file?

        def buildInfo = Artifactory.newBuildInfo()
        buildInfo.name = 'status-go'
        server.upload(uploadSpec, buildInfo)
        // TODO(adam): server upload iOS
        // TODO(adam): server upload iOS simulator
        server.publishBuildInfo(buildInfo)
    }
}
