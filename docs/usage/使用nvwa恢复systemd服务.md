## nvwa恢复systemd服务过程

dump阶段：
1. 新建一个[service_name].service.d的文件夹
2. 写入一个配置文件，该配置文件将覆盖原有service文件的配置
3. 覆盖的配置包括：ExecStart/Restart/After/User/Group属性

restore阶段：
1. systemd按照改写的ExecStart属性，将通过nvwa拉起进程
2. nvwa通过criu拉起进程
3. nvwa通过PIDFile文件，将进程的pid告知systemd，以保证systemd能够顺利接管进程
4. 删除覆盖的配置文件目录
5. 执行systemd配置reload


## 使用nvwa恢复systemd服务的限制：

1. 当前仅在Type=simple下进行了适配
2. 由于对unix socket的保存恢复支持问题，service文件需要配置StandardOutput/StandardError到某个文件
3. service文件需要有PIDFile属性


## service file example (仅用于示例，请勿直接用于现网环境)

```
[Unit]
Description=test

[Service]
ExecStart=/root/test_nvwa/test_nvwa.sh
User=root
Group=root
Type=simple
PIDFile=/root/test_nvwa_pid
StandardOutput=file:/root/log1
StandardError=file:/root/log2

[Install]
WantedBy=multi-user.target

```
