# storage-mining-tool
Storage mining tool for Polkadot Hackathon

## Features
* Automatically register to cess chain
* Mining TCESS coins
* Proof of Copy and Proof of Time and Space
* Violation punishment

## Minimum OS version requirements
CentOS_x64 8.2

## System configuration

### CentOS

##### Dependent package

```
sudo yum install wget util-linux ocl-icd-* -y

sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum makecache
sudo dnf -y  install docker-ce --nobest
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
sudo newgrp docker
```

##### Open firewall port

If the firewall is turned on, you need to open the running port.

```
$ sudo firewall-cmd --permanent --add-port=15001-15010/tcp
$ sudo firewall-cmd --reload
```

#### Parameter file
Download link：http://cess.cloud/FAQ, Article 12.
Unzip the parameter file and put it in the `/usr/cess-proof-parameters/` directory of the miner
```
$ sudo mkdir -p /usr/cess-proof-parameters
$ wget https://d2gxbb5i8u5h7r.cloudfront.net/parameterfile.zip
$ unzip -d /usr/cess-proof-parameters parameterfile.zip
$ cd /usr/cess-proof-parameters
$ mv parameterfile/v28-* .
```

### Polkadot wallet
1. Browser access:https://polkadot.js.org/apps/?rpc=wss%3A%2F%2Fcess.today%2Frpc2-hacknet%2Fws%2F#/accounts
2. Click Add Account to add two accounts. The first account is used to authenticate and operate the cess chain, and the second account is used to save income.
3. Open the faucet address:http://data.cesslab.co.uk/faucet/, enter the address of account one, and receive TCESS coins.
4. We need the public key of the account two address to issue rewards, and the public key can be obtained by converting the ss58 address online:https://polkadot.subscan.io/tools/ss58_transform

## Operation mining
1. Download the mining software package at: https://github.com/CESSProject/storage-mining-tool/releases/tag/v0.1.0
2. Modify the following configuration items in the start-mining.sh file：
```
# Path to the mounted disk where the data is saved
mountedPath=''
# Installation path of Fastdfs, you should install it in the mounted path
dfsInstallPath=''
# RPC address of CESS test chain
rpcAddr='wss://cesslab.co.uk/rpc2-hacknet/ws/'
# The CESS token  that the miner needs to pledge when registering, the unit is TCESS
pledgeTokens=2000
# Total space used to mining, the unit is GB
storageSpace=0
# The IP address of the machine's public network used by the mining program
serviceIpAddr=''
# Port number monitored by the mining program
servicePort=15001
# Port number for file service monitoring
filePort=15002
# Public key of income account
incomeAccountPubkey=''
# Phrase words or seeds for identity account
idAccountPhraseOrSeed=''
```

## Usage
* Start mining
```
$ sudo chmod +x start-mining.sh
$ sudo ./start-mining.sh
```
