def getVersion() {
  return readFile("${env.STATUS_PATH}/VERSION").trim()
}

def gitCommit() {
  return GIT_COMMIT.take(6)
}

def timestamp() {
  def now = new Date(currentBuild.timeInMillis)
  return now.format('yyMMdd-HHmmss', TimeZone.getTimeZone('UTC'))
}

def suffix() {
  if (params.RELEASE == true) {
    return readFile("${env.WORKSPACE}/${env.STATUS_PATH}/VERSION").trim()
  } else {
    return "${timestamp()}-${gitCommit()}"
  }
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

def buildBranch(name = null, buildType = null) {
  /* need to drop origin/ to match definitions of child jobs */
  def branchName = env.GIT_BRANCH.replace('origin/', '')
  /* always pass the BRANCH and BUILD_TYPE params with current branch */
  def resp = build(
    job: name,
    /* this allows us to analize the job even after failure */
    propagate: false,
    parameters: [
      [name: 'BRANCH',  value: branchName,     $class: 'StringParameterValue'],
      [name: 'RELEASE', value: params.RELEASE, $class: 'BooleanParameterValue'],
    ]
  )
  /* BlueOcean seems to not show child-build links */
  println "Build: ${resp.getAbsoluteUrl()} (${resp.result})"
  if (resp.result != 'SUCCESS') {
    error("Build Failed")
  }
  return resp
}

def copyArts(projectName, buildNo) {
  copyArtifacts(
    projectName: projectName,
    target: 'pkg',
    flatten: true,
    selector: specific("${buildNo}")
  )
}

return this

