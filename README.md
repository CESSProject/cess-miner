<h1 align="center">CESS-BUCKET</h1>

<p align="center">  
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.16-blue.svg" /></a>
  <br>
</p>

cess-bucket is a mining program provided by cess platform for storage miners.

## Building & Documentation

> Note: The default `master` branch is the main branch, please use with caution. For the latest stable version, checkout the most recent [`Latest release`](https://github.com/CESSProject/cess-bucket/releases).

For complete instructions on how to build, install and use cess-bucket, Please refer to after further improvement. Basic build instructions can be found further down in this readme.

## Reporting a Vulnerability

Please send an email to tech@cess.one.

## Basic Build Instructions

**System-specific Software Dependencies**:

Building cess-bucket requires some system dependencies, usually provided by your distribution.

**Ubuntu/Debian**:
- Ubuntu_x64 ≥ 18.04
```
sudo apt install ocl-icd-* gcc git curl hwloc libhwloc-dev wget util-linux -y && sudo apt upgrade -y
```

**RedHat/CentOS**:
- CentOS_x64 ≥ 8.2
```
sudo yum install ocl-icd-* gcc git curl hwloc libhwloc-dev wget util-linux -y && sudo apt upgrade -y
```

For other Linux distributions, please refer to the corresponding technical documentation.

#### Go

To build cess-bucket, you need a working installation of [Go 1.16.5 or higher](https://golang.org/dl/):

```bash
wget -c https://golang.org/dl/go1.16.4.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

**TIP:**
You'll need to add `/usr/local/go/bin` to your path. For most Linux distributions you can run something like:

```shell
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc && source ~/.bashrc
```

See the [official Golang installation instructions](https://golang.org/doc/install) if you get stuck.

#### firewall

**If the firewall is turned on, you need to open the running port**:

**Ubuntu/Debian**:
```
sudo ufw allow 15001:15010/tcp
```

**RedHat/CentOS**:
```
sudo firewall-cmd --permanent --add-port=15001-15010/tcp
sudo firewall-cmd --reload
```

> Note: ensure that 15001 ~ 15010 ports are released in the hardware firewall or security group policy of the server provider: since the security group policy settings provided by different servers are different, please consult the server provider

#### parameter file

Download link：http://cess.cloud/FAQ, Article 12.
Unzip the parameter file and put it in the `/usr/local/cess-proof-parameters/` directory of the miner

```
sudo mkdir -p /usr/local/cess-proof-parameters
wget https://d2gxbb5i8u5h7r.cloudfront.net/parameterfile.zip
sudo unzip -j -d /usr/local/cess-proof-parameters/ parameterfile.zip "parameterfile/*"
```

### Polkadot wallet

1. Browser access:https://polkadot.js.org/apps/?rpc=wss%3A%2F%2Fcess.today%2Frpc2-hacknet%2Fws%2F#/accounts
2. Click Add Account to add two accounts. The first account is used to authenticate and operate the cess chain, and the second account is used to save income.
3. Open the faucet address:http://data.cesslab.co.uk/faucet/, enter the address of account one, and receive TCESS coins.
4. We need the public key of the account two address to issue rewards, and the public key can be obtained by converting the ss58 address online:https://polkadot.subscan.io/tools/ss58_transform

## Build from source

1. Clone the source code to your working directory

```
git clone --recurse-submodules https://github.com/CESSProject/cess-bucket.git
cd cess-bucket
```

2. Execute the commands from [go get](https://github.com/CESSProject/cess-ffi#go-get) section

3. build mining

```
go build -o mining cmd/main/main.go
```

This will create an executable file called **'mining'**

## Usage for mining

**flag**:
| Flag      | Description                             |
| --------- | --------------------------------------- |
| -c        | Specify the configuration file location |
| -h,--help | print help information                  |

**command**:
| Command  | Description                          |
| -------- | ------------------------------------ |
| version  | print version number                 |
| default  | Generate configuration file template |
| register | Miners register to the CESS chain    |
| state    | Query the miner's own information    |
| mining   | Start mining normally                |
| exit     | Exit mining                          |
| increase | increase tokens                      |
| withdraw | Redeem tokens                        |
| obtain   | Get CESS coins from the faucet       |

## How to use mining

1. Generate configuration file template
```
sudo chmod +x mining && ./mining default
```

2. Modify the configuration file name to conf.toml and modify the following configuration items:
```
sudo mv config_template.toml conf.toml 
```

```
[CessChain]
# CESS chain address
ChainAddr = ""

[MinerData]
# Total space used to store files, the unit is GB
StorageSpace   = 0
# Path to the mounted disk where the data is saved
MountedPath    = ""
# The IP address of the machine's public network used by the mining program
ServiceAddr    = ""
# Port number monitored by the mining program
ServicePort    = 
# Public key of revenue account
RevenuePuK     = ""
# Phrase words or seeds for transaction account
TransactionPrK = ""
```

3. Register to the CESS chain
```
sudo ./mining register
```

4. Start mining normally
```
sudo nohup ./mining mining 2>&1 &
```

## License
Licensed under [Apache 2.0](https://github.com/CESSProject/cess/blob/main/LICENSE)
