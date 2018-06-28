rrpBuildGoCode {
    projectKey = 'rfid-alert-service'
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"

    infra = [
        stackName: 'RRP-CodePipeline-RFID-Alert-Service'
    ]
}