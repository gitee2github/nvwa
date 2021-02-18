# nvwa

#### 介绍

一个用于自动化openEuler热升级过程的工具

#### 构建方式

```
cd src
go get nvwa
go build
```

#### 关于配置

配置文件放置在config目录下，运行时二进制会去三个地方寻找配置文件，按照先后为：

1. 运行目录
2. 运行目录的config子目录
3. /etc/nvwa

配置文件包括:

1. nvwa-restore.yaml
    
    需要进行现场恢复的进程和服务，两者区别在于，对于每一个服务，nvwa会去修改systemd的配置，通过systemd恢复运行状态

2. nvwa-server.yaml

    热升级使用中需要用到的目录，日志，二进制目录配置等等

#### 支持的命令

+ nvwa config -- 打印server的运行信息和热更新的配置信息(待实现)

+ nvwa check -- 运行环境检查(待实现)

+ nvwa update <version> -- 热升级到相应的内核版本(相关文件需放置在/boot下)

+ nvwa restore <process> -- 恢复某个之前freeze的进程

#### 开发计划

+ 实现config和check命令

+ 支持rpm包构建