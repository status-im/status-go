def gitCommit() {
  return GIT_COMMIT.take(6)
}

def timestamp() {
  def now = new Date(currentBuild.timeInMillis)
  return now.format('yyMMdd-HHmmss', TimeZone.getTimeZone('UTC'))
}

def suffix() {
  return "${timestamp()}-${gitCommit()}"
}

def gitBranch() {
  return env.GIT_BRANCH.replace('origin/', '')
}

def getFilename(path) {
  return path.tokenize('/')[-1]
}

def getReleaseDir() {
  return '/tmp/release-' + new File(env.WORKSPACE + '/VERSION').text.trim()
}

def uploadArtifact(path) {
  /* defaults for upload */
  def domain = 'ams3.digitaloceanspaces.com'
  def bucket = 'status-go'
  withCredentials([usernamePassword(
    credentialsId: 'digital-ocean-access-keys',
    usernameVariable: 'DO_ACCESS_KEY',
    passwordVariable: 'DO_SECRET_KEY'
  )]) {
    sh """
      s3cmd \\
        --acl-public \\
        --host='${domain}' \\
        --host-bucket='%(bucket)s.${domain}' \\
        --access_key=${DO_ACCESS_KEY} \\
        --secret_key=${DO_SECRET_KEY} \\
        put ${path} s3://${bucket}/
    """
  }
  return "https://${bucket}.${domain}/${getFilename(path)}"
}

return this
