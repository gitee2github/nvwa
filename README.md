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

+ nvwa check -- 运行环境检查

+ nvwa update <version> -- 热升级到相应的内核版本(相关文件需放置在/boot下)

    nvwa将会去/boot目录下寻找需要的kernel和rootfs，kernel的命名格式需为vmlinuz-<version>, rootfs命名格式需为initramfs-<version>.img

+ nvwa restore <process> -- 恢复某个之前freeze的进程

+ nvwa help

    显示client相关的帮助信息

+ nvwa --help

    显示server相关的帮助信息

#### 开发计划

+ 将/boot目录下kernel和rootfs的命名格式放入配置项
+ 支持日志重定向至文件
+ 配置文件改变后，服务自动重新加载
+ server必须以root权限运行
+ server/client帮助文本的问题
+ 优化使用体验
+ 配置文件存在模糊的地方
+ nvwa的启动时机存在一定问题(criu对lsm之类存在依赖)
+ 网络dump/restore变成可选(否则会造成开机网络失效)