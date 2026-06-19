FROM harbor.status.im/infra/ci-build-containers:linux-base-1.0.0

# Named volume status-go-gomodcache inherits this dir's ownership on first
# creation; without jenkins ownership, go mod cache writes fail under --user jenkins.
RUN mkdir -p /home/jenkins/gomodcache && \
    chown jenkins:jenkins /home/jenkins/gomodcache
