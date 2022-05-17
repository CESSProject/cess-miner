# <h1 align="center">CESS-BUCKET &middot; [![GitHub license](https://img.shields.io/badge/license-Apache2-blue)](#LICENSE) <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.16-blue.svg" /></a></h1>

cess-bucket is a mining program provided by cess platform for storage miners.

## Building & Documentation

> Note: The default `master` branch is the main branch, please use with caution. For the latest stable version, checkout the most recent [`Latest release`](https://github.com/CESSProject/cess-bucket/releases).

For complete instructions on how to build, install and use cess-bucket, Please refer to after further improvement. Basic build instructions can be found further down in this readme.

## Reporting a Vulnerability

If you find out any vulnerability, Please send an email to tech@cess.one.
we are happy to communicate with you

## System-specific Software Dependencies

Building cess-bucket requires some system dependencies, usually provided by your distribution.

- Ubuntu/Debian(≥ 18.04):
```
sudo apt upgrade -y && sudo apt install m4 g++ flex bison make gcc git curl wget lzip util-linux -y
```

- RedHat/CentOS(≥ 8.2):
```
sudo yum upgrade -y && sudo dnf install m4 flex bison -y && sudo yum install gcc-c++ git curl wget lzip util-linux -y
```

For other Linux distributions, please refer to the corresponding technical documentation.

### Installing pbc library
```
sudo wget https://gmplib.org/download/gmp/gmp-6.2.1.tar.lz
sudo lzip -d gmp-6.1.2.tar.lz
sudo tar -xvf gmp-6.1.2.tar
sudo cd gmp-6.2.1/
sudo chmod +x ./configure
sudo ./configure --enable-cxx
sudo make
sudo make check
sudo make install
cd ..

sudo wget https://crypto.stanford.edu/pbc/files/pbc-0.5.14.tar.gz
sudo tar -zxvf pbc-0.5.14.tar.gz
sudo cd pbc-0.5.14/
sudo chmod +x ./configure
sudo ./configure
sudo make
sudo make install
touch /etc/ld.so.conf.d/libpbc.conf
test -s /etc/ld.so.conf.d/libpbc.conf && sed -i "\$a /usr/local/lib" /etc/ld.so.conf.d/libpbc.conf || echo "/usr/local/lib" >> /etc/ld.so.conf.d/libpbc.conf
sudo ldconfig
```

### Firewall configuration

If the firewall is turned on, you need to open the running port, The default port is 15001.

- Ubuntu/Debian**:
```
sudo ufw allow 15001/tcp
```

- RedHat/CentOS**:
```
sudo firewall-cmd --permanent --add-port=15001/tcp
sudo firewall-cmd --reload
```

## Go language environment

To build cess-bucket, you need a working installation of [Go 1.16.5 or higher](https://golang.org/dl/):

```bash
wget -c https://golang.org/dl/go1.16.5.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

**TIP:**
You'll need to add `/usr/local/go/bin` to your path. For most Linux distributions you can run something like:

```shell
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc && source ~/.bashrc
```

See the [official Golang installation instructions](https://golang.org/doc/install) if you get stuck.

> Note: ensure that 15001 ports are released in the hardware firewall or security group policy of the server provider: since the security group policy settings provided by different servers are different, please consult the server provider
> 

## Polkadot wallet

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

2. build mining

```
go build -o bucket cmd/main/main.go
```

This will create an executable file called **'bucket'**

## Usage for bucket

**flag**:
| Flag      | Description                             |
| --------- | --------------------------------------- |
| -c        | Specify the configuration file location |
| -h,--help | print help information                  |

**command**:
| Command  | Description                                    |
| -------- | ---------------------------------------------- |
| version  | print version number                           |
| default  | Generate configuration file template           |
| register | Register mining miner information to the chain |
| state    | Query mining miner information                 |
| run      | Start mining normally                          |
| exit     | Exit the mining platform                       |
| increase | Increase the deposit of mining miner           |
| withdraw | Redemption deposit of mining miner             |
| obtain   | Get the test coins used by the testnet         |

## How to use mining
1. Generate configuration file template
```
sudo chmod +x bucket && ./bucket default
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
StorageSpace   = 1000
# Path to the mounted disk where the data is saved
MountedPath    = ""
# The IP address of the machine's public network used by the mining program
ServiceAddr    = ""
# Port number monitored by the mining program
ServicePort    = 15001
# Public key of revenue account
RevenueAcc     = ""
# Phrase words or seeds for signature account
SignaturePrk = ""
```

3. Register to the CESS chain
```
sudo ./bucket register
```

4. Start bucket normally
```
sudo nohup ./bucket run 2>&1 &
```


## License
Licensed under [Apache 2.0](https://github.com/CESSProject/cess-bucket/blob/main/LICENSE)
