def getVersion() {
  return readFile("${env.STATUS_PATH}/VERSION").trim()
}

def gitCommit() {
  return GIT_COMMIT.take(6)
}

def parentOrCurrentBuild() {
  def c = currentBuild.rawBuild.getCause(hudson.model.Cause$UpstreamCause)
  if (c == null) { return currentBuild }
  return c.getUpstreamRun()
}

def timestamp() {
  def now = new Date(parentOrCurrentBuild().timeInMillis)
  return now.format('yyMMdd-HHmmss', TimeZone.getTimeZone('UTC'))
}

def suffix() {
  if (params.RELEASE == true) {
    return getVersion()
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

def copyArts(build) {
  /**
   * The build argument is of class RunWrapper.
   * https://javadoc.jenkins.io/plugin/workflow-support/org/jenkinsci/plugins/workflow/support/steps/build/RunWrapper.html
   **/
  copyArtifacts(
    projectName: build.fullProjectName,
    target: 'pkg',
    flatten: true,
    selector: specific("${build.number}")
  )
}

return this
