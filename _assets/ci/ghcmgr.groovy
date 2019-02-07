import groovy.json.JsonBuilder

def getRepoName() {
  if (env.GIT_URL =~ /https?:\/\/[^\/]\/(.+)(\.git)?/) {
    return m.group(1)
  }
  return env.GIT_URL.split('/').last().minus('.git')
}

/* if job was started by a parent we can access it's env */
def getParentRunEnv(name) {
  def c = currentBuild.rawBuild.getCause(hudson.model.Cause$UpstreamCause)
  if (c == null) { return null }
  return c.getUpstreamRun().getEnvironment()[name]
}

/* returns duration of build in rounded up minutes */
def buildDuration() {
  def duration = currentBuild.durationString
  return '~' + duration.take(duration.lastIndexOf(' and counting'))
}

/* CHANGE_ID can be provided via the build parameters or from parent */
def changeId() {
  def changeId = env.CHANGE_ID
  changeId = params.CHANGE_ID ? params.CHANGE_ID : changeId
  changeId = getParentRunEnv('CHANGE_ID') ? getParentRunEnv('CHANGE_ID') : changeId
  if (!changeId) {
    println('This build is not related to a PR, CHANGE_ID missing.')
    println('GitHub notification impossible, skipping...')
    return null
  }
  return changeId
}

/* assemble build object valid for ghcmgr */
def buildObj(success) {
  return [
    id: env.BUILD_DISPLAY_NAME,
    commit: GIT_COMMIT.take(8),
    success: success != null ? success : true,
    platform: env.BUILD_PLATFORM,
    duration: buildDuration(),
    url: currentBuild.absoluteUrl,
    pkg_url: env.PKG_URL,
  ]
}

/**
 * This is our own service for avoiding comment spam.
 * https://github.com/status-im/github-comment-manager
 **/
def postBuild(success) {
  def changeId = changeId()
  if (changeId == null) { return } /* not in a PR build */
  def ghcmgrUrl = 'https://ghcmgr.status.im'
  def body = buildObj(success)
  def json = new JsonBuilder(body).toPrettyString()
  withCredentials([usernamePassword(
    credentialsId:  'ghcmgr-auth',
    usernameVariable: 'GHCMGR_USER',
    passwordVariable: 'GHCMGR_PASS'
  )]) {
    sh """
      curl --silent --verbose -XPOST --data '${json}' \
        -u '${GHCMGR_USER}:${GHCMGR_PASS}' \
        -H "content-type: application/json" \
        '${ghcmgrUrl}/builds/${getRepoName()}/${changeId}'
    """
  }
}

return this
