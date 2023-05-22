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

**Step 1:** Install go locale

CESS-Bucket requires [Go 1.19](https://golang.org/dl/) or higher.

See the [official Golang installation instructions](https://golang.org/doc/install).

**Step 2:** Build a bucket

```shell
git clone https://github.com/CESSProject/cess-bucket.git
cd cess-bucket/
go build -o bucket cmd/main.go
chmod +x bucket
```

If all goes well, you will get a mining program called `bucket`.

## Configure Wallet

**Step 1:** Register two cess wallet

For wallet one, it is called an  `earnings account`, which is used to receive rewards from mining, and you should keep the private key carefully.

For wallet two, it is called a `staking account` and is used to staking some tokens and sign blockchain transactions.

Please refer to [Create-CESS-Wallet](https://github.com/CESSProject/cess/wiki/Create-a-CESS-Wallet) to create your cess wallet.

**Step 2:** Recharge your signature account

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

| Command  | Description                                    |
| -------- | ---------------------------------------------- |
| version  | Print version number                           |
| config   | Generate configuration file                    |
| register | Register mining miner information to the chain |
| stat     | Query storage miner information                |
| run      | Automatically register and run                 |
| exit     | Unregister the storage miner role              |
| increase | Increase the stakes of storage miner           |
| withdraw | Withdraw stakes                                |
| update   | Update inforation                              |

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
2023/05/22 23:21:47 👉 Please enter the rpc address of the chain, multiple addresses are separated by spaces:
wss://testnet-rpc0.cess.cloud/ws/ wss://testnet-rpc1.cess.cloud/ws/
2023/05/22 23:21:55 👉 Please enter the workspace, press enter to use / by default workspace:
/
2023/05/22 23:21:57 👉 Please enter your earnings account, if you are already registered and do not want to update, please press enter to skip:
cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y
2023/05/22 23:22:06 👉 Please enter your service port:
15001
2023/05/22 23:22:09 👉 Please enter the maximum space used by the storage node in GiB:
2000
2023/05/22 23:22:12 👉 Please enter the mnemonic of the staking account:

```

**method two**

```
# ./bucket run --rpc wss://testnet-rpc0.cess.cloud/ws/,wss://testnet-rpc1.cess.cloud/ws/ --ws / --earnings cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y --port 15001 --space 2000
2023/05/22 23:29:44 👉 Please enter the mnemonic of the staking account:
```

**Background operation mode**
Generate configuration file:
```
./bucket config
2023/05/22 23:42:00 ✅ /root/bucket/conf.yaml
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

- Query miner status

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

- Increase the miner's deposit by 1000
```shell
./bucket increase 1000
```

- Update the miner's earnings account
```shell
./bucket update earnings <earnings account>
```

## License
Licensed under [Apache 2.0](https://github.com/CESSProject/cess-bucket/blob/main/LICENSE)
