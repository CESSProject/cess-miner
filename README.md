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

### Install pbc library
```
sudo wget https://gmplib.org/download/gmp/gmp-6.2.1.tar.lz
sudo lzip -d gmp-6.2.1.tar.lz
sudo tar -xvf gmp-6.2.1.tar
cd gmp-6.2.1/
sudo chmod +x ./configure
sudo ./configure --enable-cxx
sudo make
sudo make check
sudo make install
cd ..

sudo wget https://crypto.stanford.edu/pbc/files/pbc-0.5.14.tar.gz
sudo tar -zxvf pbc-0.5.14.tar.gz
cd pbc-0.5.14/
sudo chmod +x ./configure
sudo ./configure
sudo make
sudo make install
sudo touch /etc/ld.so.conf.d/libpbc.conf
sudo test -s /etc/ld.so.conf.d/libpbc.conf
sudo sed -i "\$a /usr/local/lib" /etc/ld.so.conf.d/libpbc.conf || echo "/usr/local/lib" >> /etc/ld.so.conf.d/libpbc.conf
sudo ldconfig
```

### Firewall configuration

If the firewall is turned on, you need to open the running port, The default port is 15001.

- Ubuntu/Debian
```
sudo ufw allow 15001/tcp
```
- RedHat/CentOS
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

1. Browser access: [App](https://testnet-rpc.cess.cloud/explorer) implemented by [CESS Explorer](https://github.com/CESSProject/cess-explorer).
2. Click Add Account to add two accounts. The first account is used to authenticate and operate the cess chain, and the second account is used to save income.
3. The way to claim tokens will be made public in the near future.

## Build from source

Clone the code and build:
```
git clone https://github.com/CESSProject/cess-bucket.git
cd cess-bucket
go build -o bucket cmd/main/main.go
```

This will create an executable file called **'bucket'**

## Usage for bucket

**flag**:
| Flag        | Description                             |
| ----------- | --------------------------------------- |
| -c,--config | Custom profile |
| -h,--help   | Print help information                  |

**command**:
| Command  | Description                                    |
| -------- | ---------------------------------------------- |
| version  | Print version number                           |
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
# The rpc address of the chain node
RpcAddr      = ""
# Path to the mounted disk where the data is saved
MountedPath  = ""
# Total space used to store files, the unit is GB
StorageSpace = 1000
# The IP address of the machine's public network used by the mining program
ServiceAddr  = ""
# Port number monitored by the mining program
ServicePort  = 15001
# The address of income account
IncomeAcc    = ""
# phrase or seed of the signature account
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
