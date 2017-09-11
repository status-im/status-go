node {
    stage('Debug') {
        sh 'env'
    }

    stage('Build') {
        def scmVars = checkout scm
        print scmVars

        // sh 'make statusgo-android'
    }

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
