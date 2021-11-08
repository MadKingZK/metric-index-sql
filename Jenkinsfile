node('master') {
    def root = tool type: 'go', name: 'go15'
    
    // Export environment variables pointing to the directory where Go was installed
    stage('meke Env & Build') {
        withEnv(["GOROOT=${root}", "PATH+GO=${root}/bin"]) {
            sh 'go version'
            sh 'go mod tidy'
            sh 'env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build --tags=jsoniter -o metric-index main.go'
        }
    }
}
