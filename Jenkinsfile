node {
    stage('Debug') {
        sh 'env'
    }

    stage('Build') {
        sh 'make statusgo-android'
    }

    stage('Deploy') {
        withCredentials([usernameColonPassword(credentialsId: 'artifactory-deploy-bot', variable: 'USERPASS')]) {
            sh '''
                set +x

                version=$(echo "${GIT_BRANCH##origin/}-g${GIT_COMMIT:0:7}" | tr / -)
                curl -u "${USERPASS}" \
                    -X PUT "http://139.162.11.12:8081/artifactory/libs-release-local/status-im/status-go/${version}/status-go-${version}.aar" \
                    -T ./build/bin/statusgo-android-16.aar
            '''
        }
    }
}
