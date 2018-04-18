sudo docker secret rm brokerSecureKey
sudo docker secret rm brokerSecureCert
sudo docker secret rm mongoAuth
sudo docker secret rm connectionString
sudo docker secret rm cssdkPasswd
sudo docker secret rm gatewayCredentials
sudo docker secret create brokerSecureKey secrets/./server.key
sudo docker secret create brokerSecureCert secrets/./server.crt
sudo docker secret create mongoAuth secrets/./mongo_auth.json
sudo docker secret create connectionString secrets/./connectionString
sudo docker secret create cssdkPasswd secrets/./cssdkPasswd
sudo docker secret create gatewayCredentials secrets/./gatewayCredentials