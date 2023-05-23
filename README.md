# <h1 align="center">CESS-BUCKET &middot; [![GitHub license](https://img.shields.io/badge/license-Apache2-blue)](#LICENSE) <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.19-blue.svg" /></a> [![Go Reference](https://pkg.go.dev/badge/github.com/CESSProject/cess-bucket/edit/main/README.md.svg)](https://pkg.go.dev/github.com/CESSProject/cess-bucket/edit/main/README.md)</h1>

CESS-Bucket is a mining program provided by cess platform for storage miners.


## Reporting a Vulnerability

If you find out any vulnerability, Please send an email to tech@cess.one, we are happy to communicate with you.


## System Requirements

- Linux-amd64


## System dependencies

Take the ubuntu distribution as an example:

```shell
sudo apt update && sudo apt upgrade
sudo apt install g++ make gcc git curl wget vim screen util-linux -y
```

## System configuration

- Firewall

If the firewall is turned on, you need to open the running port, the default port is 15001.

Take the ubuntu distribution as an example:

```shell
sudo ufw allow 15001/tcp
```
- Network optimization (optional)

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

## Build from source

**Step 1:** Install go

CESS-Bucket requires [Go 1.19](https://golang.org/dl/) or higher, See the [official Golang installation instructions](https://golang.org/doc/install).

Open gomod mode:
```
go env -w GO111MODULE="on"
```

Users in China can add go proxy to speed up the download:
```
go env -w GOPROXY="https://goproxy.cn,direct"
```

**Step 2:** Clone code

```shell
git clone https://github.com/CESSProject/cess-bucket.git
```

**Step 3:** Build a bucket

```shell
cd cess-bucket/
go build -o bucket cmd/main.go
```

**Step 4:** Grant execute permission

```shell
chmod +x bucket
```

If all goes well, you will get a mining program called `bucket`.

## Configure Wallet

**Step 1:** Register two cess wallet

For wallet one, it is called an  `earnings account`, which is used to receive rewards from mining, and you should keep the private key carefully.

For wallet two, it is called a `staking account` and is used to staking some tokens and sign blockchain transactions.

Please refer to [Create-CESS-Wallet](https://github.com/CESSProject/cess/wiki/Create-a-CESS-Wallet) to create your cess wallet.

**Step 2:** Recharge your staking account

The staking amount is calculated based on the space you configure. The minimum staking amount is 2000CESS, and an additional 2000CESS staking is required for each additional 1TiB of space.

If you are using the test network, Please join the [CESS discord](https://discord.gg/mYHTMfBwNS) to get it for free. If you are using the official network, please buy CESS tokens.

## View bucket features

The `bucket` has many functions, you can use `-h` or `--help` to view, as follows:

- Flags

| Flag        | Description                                        |
| ----------- | -------------------------------------------------- |
| -c,--config | custom configuration file (default "conf.yaml")    |
| -h,--help   | help for bucket                                    |
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
| increase |            | Increase the stakes of storage miner           |
| withdraw |            | Withdraw stakes                                |
| update   | earnings   | Update earnings account                        |

## Start mining
The bucket program has two running modes: foreground and background.

**Foreground operation mode**

The foreground operation mode requires the terminal window to be kept all the time, and the window cannot be closed. You can use the screen command to create a window for the bucket and ensure that the window always exists. 
Create and enter the bucket window command:
```
screen -S bucket
```
Press `ctrl + A + D` to exit the bucket window without closing it.

View window list command:
```
screen -ls
```
Re-enter the bucket window command:
```
screen -r bucket
```


**method one**

Enter the `bucket run` command to run directly, and enter the information according to the prompt to complete the startup:

```
# ./bucket run
>> Please enter the rpc address of the chain, multiple addresses are separated by spaces:
wss://testnet-rpc0.cess.cloud/ws/ wss://testnet-rpc1.cess.cloud/ws/
>> Please enter the workspace, press enter to use / by default workspace:
/
>> Please enter your earnings account, if you are already registered and do not want to update, please press enter to skip:
cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y
>> Please enter your service port:
15001
>> Please enter the maximum space used by the storage node in GiB:
2000
>> Please enter the mnemonic of the staking account:

OK /cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw/bucket
```

**method two**

```
# ./bucket run --rpc wss://testnet-rpc0.cess.cloud/ws/,wss://testnet-rpc1.cess.cloud/ws/ --ws / --earnings cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y --port 15001 --space 2000
>> Please enter the mnemonic of the staking account:

OK /cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw/bucket
```

**Background operation mode**

Generate configuration file:
```
./bucket config
OK /root/bucket/conf.yaml
```
Edit the configuration file and fill in the correct information, then run:
```
nohup ./bucket run -c /root/bucket/conf.yaml &
```
If the configuration file is named conf.yaml and is located in the same directory as the bucket program, you can specify without -c:
```
nohup ./bucket run &
```

## Other commands

The examples in this chapter use the conf.yaml configuration file in the current directory by default.

- stat
```shell
./bucket stat
+------------------+------------------------------------------------------+
| peer id          | 12D3KooWSEX3UkyU2R6S1wERs4iH7yp2yVCWX2YkReaokvCg7uxU |
| state            | positive                                             |
| staking amount   | 2400 TCESS                                           |
| validated space  | 1023410176 bytes                                     |
| used space       | 25165824 bytes                                       |
| locked space     | 0 bytes                                              |
| staking account  | cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw    |
| earnings account | cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y    |
+------------------+------------------------------------------------------+
```

- increase
```shell
./bucket increase 1000
OK 0xe098179a4a668690f28947d20083014e5a510b8907aac918e7b96efe1845e053
```

- update earnings
```shell
./bucket update earnings cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw
OK 0x0fa67b89d9f8ff134b45e4e507ccda00c0923d43c3b8166a2d75d3f42e5a269a
```

- version
```shell
./bucket version
bucket v0.6.0
```

- exit
```shell
./bucket exit
OK 0xf6e9573ba53a90c4bbd8c3784ef97bbf74bdb1cf8c01df697310a64c2a7d4513
```

- withdraw
```shell
./bucket withdraw
OK 0xfbcc77c072f88668a83f2dd3ea00f3ba2e5806aae8265cfba1582346d6ada3f1
```

## License
Licensed under [Apache 2.0](https://github.com/CESSProject/cess-bucket/blob/main/LICENSE)
