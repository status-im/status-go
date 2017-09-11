def version(branch, sha) {
    return branch.replaceAll(/\//, '-') + '-' + sha
}

node {
    def remoteOriginRegex = ~/^remotes\/origin\//

    gitSHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
    gitShortSHA = gitSHA.take(7)
    String gitBranch = sh(returnStdout: true, script: 'git name-rev --name-only HEAD').trim() - remoteOriginRegex

    stage('Debug') {
        sh 'env'
        println(gitBranch)
        println(gitSHA)
        println(version(gitBranch, gitShortSHA))
    }

    // stage('Build') {
    //     sh 'make statusgo-android'
    // }

    // stage('Deploy') {
    //     withCredentials([usernameColonPassword(credentialsId: 'artifactory-deploy-bot', variable: 'USERPASS')]) {
    //         gitSHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
    //         gitShortSHA = gitCommit.take(7)
    //         gitBranch = sh(returnStdout: true, script: 'git rev-parse --abbrev-ref HEAD').trim()

    //         sh '''
    //             set +x

    //             version=$(echo "${gitBranch##origin/}-${gitShortSHA}" | tr / -)
    //             curl -u "${USERPASS}" \
    //                 -X PUT "http://139.162.11.12:8081/artifactory/libs-release-local/status-im/status-go/${version}/status-go-${version}.aar" \
    //                 -T ./build/bin/statusgo-android-16.aar
    //         '''
    //     }
    // }
}
