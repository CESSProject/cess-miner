# storage-mining-tool
Storage mining tool for Polkadot Hackathon

## Feature
* Automatically register to cess chain
* Mining TCESS coins
* Proof of Copy and Proof of Time and Space
* Violation punishment

## Minimum system version
CentOS_x64 8.2

## system configuration

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
Download link：http://cess.today/FAQ, Article 12.
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
3. Open the faucet address, enter the address of account one, and receive TCESS coins.
4. We need the public key of the account two address to issue rewards, and the public key can be obtained by converting the ss58 address online:https://polkadot.subscan.io/tools/ss58_transform

## Operation mining

* View help information for the node program

```
$ sudo chmod +x ./cessmining
$ sudo ./cessmining -h 
CESS-Storage-Mining

Usage:
    ./cessmining [arguments] [file]

Arguments:
  -c configuration file
        Run the program directly to generate
        Specify the configuration file to ensure that the program runs correctly
  -h    Print Help (this message) and exit
  -v    Print version information and exit
```

* Configuration file description
Running the mining program will generate the `conf_default.toml` configuration file by default
```
[cessChain]
# RPC address of CES public chain
rpcAddr = "wss://cess.today/rpc2-hacknet/ws/"

[minerData]
# The cess coin that the miner needs to pledge when registering, the unit is TCESS.
pledgeTokens                 = 2000
# Total space used to store files, the unit is GB.
storageSpaceForCessMining    = 1024
# Path to the mounted disk where the data is saved
mountedPathForCessMining     = "/"
# The IP address of the machine's public network used by the mining program.
serviceIpAddress             = ""
# Port number monitored by the mining program.
servicePort                  = "15001"
# Public key of income account.
incomeAccountPublicKey       = ""
# Phrase words or seeds for identity accounts.
identifyAccountPhraseOrSeed  = ""
```

## Usage

* Start mining
```
$ sudo ./cessmining -c conf.toml
```
