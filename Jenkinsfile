#!/usr/bin/env groovy

@NonCPS
def getVersion(branch, sha) {
    if (sha?.trim()) {
        return branch.replaceAll(/\//, '-') + '-' + sha
    }

    return branch.replaceAll(/\//, '-')
}

node {
    checkout scm

    def remoteOriginRegex = ~/^remotes\/origin\//

    gitSHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
    gitShortSHA = gitSHA.take(7)
    gitBranch = sh(returnStdout: true, script: 'git name-rev --name-only HEAD').trim() - remoteOriginRegex

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
        parallel (
            'statusgo-android': {
                sh 'make statusgo-android'
            },
            'statusgo-ios-simulator': {
                sh '''
                    make statusgo-ios-simulator
                    cd build/bin/statusgo-ios-9.3-framework/
                    zip -r status-go-ios.zip Statusgo.framework
                '''
            }
        )
    }

    stage('Deploy') {
        // For branch builds, replace the old artifact. For develop keep all of them.
        def version = gitBranch == 'develop' ? getVersion(gitBranch, gitShortSHA) : getVersion(gitBranch, '')

        // Clean up the previous artifacts.
        withCredentials([usernameColonPassword(credentialsId: 'artifactory-deploy-bot', variable: 'USERPASS')]) {
            sh """
                curl -u "${USERPASS}" -X DELETE "http://artifacts.status.im:8081/artifactory/libs-release-local/status-im/status-go/${version}"
                curl -u "${USERPASS}" -X DELETE "http://artifacts.status.im:8081/artifactory/libs-release-local/status-im/status-go-ios-simulator/${version}"
            """
        }

        def server = Artifactory.server 'artifactory'
        def uploadSpec = """{
            "files": [
                {
                    "pattern": "build/bin/statusgo-android-16.aar",
                    "target": "libs-release-local/status-im/status-go/${version}/status-go-${version}.aar"
                },
                {
                    "pattern": "build/bin/statusgo-ios-9.3-framework/status-go-ios.zip",
                    "target": "libs-release-local/status-im/status-go-ios-simulator/${version}/status-go-ios-simulator-${version}.zip"
                }
            ]
        }"""

        def buildInfo = Artifactory.newBuildInfo()
        buildInfo.env.capture = false
        buildInfo.name = 'status-go'
        server.upload(uploadSpec, buildInfo)
        server.publishBuildInfo(buildInfo)
    }
}
