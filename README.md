# <h1 align="center">CESS-BUCKET </br> [![GitHub license](https://img.shields.io/badge/license-Apache2-blue)](#LICENSE) <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.20-blue.svg" /></a> [![Go Reference](https://pkg.go.dev/badge/github.com/CESSProject/cess-miner/edit/main/README.md.svg)](https://pkg.go.dev/github.com/CESSProject/cess-miner/edit/main/README.md) [![build](https://github.com/CESSProject/cess-miner/actions/workflows/build.yml/badge.svg)](https://github.com/CESSProject/cess-miner/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/CESSProject/cess-miner)](https://goreportcard.com/report/github.com/CESSProject/cess-miner)</h1>

CESS-Bucket is a mining program provided by cess platform for storage miners.

## 📝 Reporting a Vulnerability
If you find any system errors or you have better suggestions, please submit an issue or submit a pull request. You can also join the [CESS discord](https://discord.gg/mYHTMfBwNS) to communicate with us.

## 📢 Announcement
**CESS test network rpc endpoints**
```
wss://testnet-rpc.cess.network/ws/
```
**CESS test network bootstrap node**
```
_dnsaddr.boot-miner-testnet.cess.network
```

## 🚰 CESS test network faucet
```
https://www.cess.network/faucet.html
```

## ⚠ Attention
The following commands are executed with root privileges, if the prompt `Permission denied` appears, you need to switch to root privileges, or add `sudo` at the top of these commands.

## ⚙ System configuration
### System requirements
- Linux 64-bit Intel/AMD

### Install application tools

For the Debian and  ubuntu families of linux systems:

```bash
# apt install git curl wget vim util-linux -y
```

For the Fedora, RedHat and CentOS families of linux systems:

```bash
# yum install git curl wget vim util-linux -y
```

### Firewall configuration
By default, cess-miner uses port `4001` to listen for incoming connections, if your platform blocks these two ports by default, you may need to enable access to these port.

#### ufw
For hosts with ufw enabled (Debian, Ubuntu, etc.), you can use the ufw command to allow traffic to flow to specific ports. Use the following command to allow access to a port:
```bash
# ufw allow 4001
```

#### firewall-cmd
For hosts with firewall-cmd enabled (CentOS), you can use the firewall-cmd command to allow traffic on specific ports. Use the following command to allow access to a port:
```bash
# firewall-cmd --get-active-zones
```
This command gets the active zone(s). Now, apply port rules to the relevant zones returned above. For example if the zone is public, use
```bash
# firewall-cmd --zone=public --add-port=4001/tcp --permanent
```
Note that permanent makes sure the rules are persistent across firewall start, restart or reload. Finally reload the firewall for changes to take effect.
```bash
# firewall-cmd --reload
```

#### iptables
For hosts with iptables enabled (RHEL, CentOS, etc.), you can use the iptables command to enable all traffic to a specific port. Use the following command to allow access to a port:
```
iptables -A INPUT -p tcp --dport 4001 -j ACCEPT
service iptables restart
```

### Network optimization (optional)

```shell
sysctl -w net.ipv4.tcp_syncookies = 1
sysctl -w net.ipv4.tcp_tw_reuse = 1
sysctl -w net.ipv4.tcp_tw_recycle = 1
sysctl -w net.ipv4.tcp_fin_timeout = 30
sysctl -w net.ipv4.tcp_max_syn_backlog = 8192
sysctl -w net.ipv4.tcp_max_tw_buckets = 6000
sysctl -w net.ipv4.tcp_timestsmps = 0
sysctl -w net.ipv4.ip_local_port_range = 10000 65500
```

## 💰 Configure CESS wallet

**1) Register two cess wallet**

For wallet one, it is called an  `earnings account`, which is used to receive rewards from mining, and you should keep the private key carefully.

For wallet two, it is called a `staking account` and is used to staking some tokens and sign blockchain transactions.

Please refer to [Create-CESS-Wallet](https://github.com/CESSProject/cess/wiki/Create-a-CESS-Wallet) to create your cess wallet.

**2) Recharge your staking account**

The staking amount is calculated based on the space you configure. The minimum staking amount is 4000CESS, and an additional 4000CESS staking is required for each additional 1TiB of space.

If you are using the test network, Please join the [CESS discord](https://discord.gg/mYHTMfBwNS) to get it for free. If you are using the official network, please buy CESS tokens.

## 🏗 Get the binary program
### Method one
Download the latest release of the binary application directly at：
```bash
# wget https://github.com/CESSProject/cess-miner/releases/download/v0.7.11/bucket0.7.11.linux-amd64.tar.gz
```
### Method two
Compile the binary program from the storage node source code and follow the process as follows:

**1) install go**
CESS-Bucket requires [Go 1.21](https://golang.org/dl/), See the [official Golang installation instructions](https://golang.org/doc/install).

Open gomod mode:
```bash
# go env -w GO111MODULE="on"
```

Users in China can add go proxy to speed up the download:
```bash
# go env -w GOPROXY="https://goproxy.cn,direct"
```

**2) clone code**

```bash
# git clone https://github.com/CESSProject/cess-miner.git
```

**3) compile code**

```bash
# cd cess-miner/
# go build -o miner cmd/main.go
```

**4) Grant execute permission**

```bash
# chmod +x miner
```

**5) View miner features (optional)**

The `miner` has many functions, you can use `./miner -h` or `./miner --help` to view, as follows:

- Flags

| Flag        | Description                                        |
| ----------- | -------------------------------------------------- |
| -c,--config | custom configuration file (default "conf.yaml")    |
| -h,--help   | help for cess-miner                                |
| --earnings  | earnings account                                   |
| --port      | listening port                                     |
| --rpc       | rpc endpoint list                                  |
| --space     | maximum space used(GiB)                            |
| --ws        | workspace                                          |

- Available Commands

| Command  | Subcommand | Description                                    |
| -------- | ---------- | ---------------------------------------------- |
| version  |            | Print version number                           |
| config   |            | Generate configuration file                    |
| stat     |            | Query storage miner information                |
| run      |            | Automatically register and run                 |
| exit     |            | Unregister the storage miner role              |
| increase | staking    | Increase the staking                           |
| increase | space      | Increase the declaration space                 |
| withdraw |            | Withdraw stakes                                |
| update   | earnings   | Update earnings account                        |
| reward   |            | Query reward information                       |
| claim    |            | Claim reward                                   |

## 🟢 Start mining
The miner program has two running modes: foreground and background.

> :warning: If you are not running the `miner` program with root privileges, make sure that the user you are currently logged in to has all permissions for the workspace directory you have configured. If you are logged in as `user`, the configured directory is `/cess`, and your signature account is `cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y`, execute the following command to grant permissions:
```bash
# chown -R  user:user /cess/cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y/
```

### Foreground operation mode

The foreground operation mode requires the terminal window to be kept all the time, and the window cannot be closed. You can use the screen command to create a window for the miner and ensure that the window always exists. 
Create and enter the miner window command:
```bash
# screen -S miner
```
Press `ctrl + A + D` to exit the miner window without closing it.

View window list command:
```bash
# screen -ls
```
Re-enter the miner window command:
```bash
# screen -r miner
```


**method one**

Enter the `miner run` command to run directly, and enter the information according to the prompt to complete the startup:

```bash
# ./miner run
>> Please enter the rpc address of the chain, multiple addresses are separated by spaces:
wss://testnet-rpc.cess.network/ws/
>> Please enter the workspace, press enter to use / by default workspace:
/
>> Please enter your earnings account, if you are already registered and do not want to update, please press enter to skip:
cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y
>> Please enter your service port:
15001
>> Please enter the maximum space used by the storage node in GiB:
2000
>> Please enter the mnemonic of the staking account:
*******************************************************************************
```

**method two**

```bash
# ./miner run --rpc wss://testnet-rpc.cess.network/ws/ --ws / --earnings cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y --port 4001 --space 2000
>> Please enter the mnemonic of the staking account:
*******************************************************************************
```

### Background operation mode

Generate configuration file:
```bash
# ./miner config
OK /root/miner/conf.yaml
```
Edit the configuration file and fill in the correct information, then run:
```bash
# nohup ./miner run -c /root/miner/conf.yaml &
```
If the configuration file is named conf.yaml and is located in the same directory as the miner program, you can specify without -c:
```bash
# nohup ./miner run &
```

## 💡 Other commands

- stat
```bash
# ./miner stat --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
+-------------------+------------------------------------------------------+
| name              | storage miner                                        |
| peer id           | 12D3KooWSEX3UkyU2R6S1wERs4iH7yp2yVCWX2YkReaokvCg7uxU |
| state             | positive                                             |
| staking amount    | 2400 TCESS                                           |
| staking start     | 3123                                                 |
| debt amount       | 0 TCESS                                              |
| declaration space | 1.00 TiB                                             |
| validated space   | 1.00 GiB                                             |
| used space        | 25.00 MiB                                            |
| locked space      | 0 bytes                                              |
| signature account | cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw    |
| staking account   | cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw    |
| earnings account  | cXjeCHQW3totBGhQXdAUAqjCNqk1NhiR3UK37czSeUak2pqGV    |
+-------------------+------------------------------------------------------+
```

- increase staking
```bash
# ./miner increase staking 1000000000000000000000 --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
OK 0xe098179a4a668690f28947d20083014e5a510b8907aac918e7b96efe1845e053
```

- increase space
```bash
# ./miner increase space 10 --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
OK 0xe098179a4a668690f28947d20083014e5a510b8907aac918e7b96efe1845e053
```

- update earnings
```bash
# ./miner update earnings cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
OK 0x0fa67b89d9f8ff134b45e4e507ccda00c0923d43c3b8166a2d75d3f42e5a269a
```

- version
```shell
./miner version
miner v0.7.11
```

- exit
```bash
# ./miner exit --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
OK 0xf6e9573ba53a90c4bbd8c3784ef97bbf74bdb1cf8c01df697310a64c2a7d4513
```

- withdraw
```bash
# ./miner withdraw --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
OK 0xfbcc77c072f88668a83f2dd3ea00f3ba2e5806aae8265cfba1582346d6ada3f1
```

- claim
```bash
# ./miner claim --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
OK 0x59096fd095b66665c838f89ae4f1384ab31255cdc9c80003b05b50124cfdcfee
```

- reward
```bash
# ./miner reward --rpc wss://testnet-rpc.cess.network/ws/
>> Please enter the mnemonic of the staking account:
*******************************************************************************
+------------------+---------------------------+
| total reward     | 2_613_109_650_924_024_640 |
| claimed reward   | 534_235_750_855_578_370   |
| unclaimed reward | 0                         |
+------------------+---------------------------+
```

## License
Licensed under [Apache 2.0](https://github.com/CESSProject/cess-miner/blob/main/LICENSE)
