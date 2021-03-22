## nvwa保存和恢复现场过程

在执行nvwa update xxx命令后，nvwa将读取nvwa-restore.yaml文件，获取到所有将要保存的进程信息和服务信息。

对于每一个进程，nvwa将直接借助criu生成相应的现场信息。

对于每一个服务，nvwa首先借助criu生成相应的现场信息。接着将会在/usr/systemd/system目录下，对相应的service进行一个属性设置。包括:
1. 设置ExecStart为nvwa restore service <pid>，这一设置是为了后续systemd在拉起服务过程时，能正确恢复服务的状态。
2. 设置User和Group为root，这一设置是为了保证在恢复现场过程中，nvwa有足够的权限。
3. 设置Restart为no，这一设置是为了保证进程因为保存现场而被kill后，不会重新启动。
4. 设置服务在nvwa之后启动

在所有现场都正确保存后，nvwa通过kexec进行内核切换。等到新内核启动时，nvwa作为一个程序启动，nvwa根据配置文件，去检查所有进程的现场信息，如果存在，则通过criu进行恢复。

对于服务来说，当systemd加载此类服务时，由于ExecStart属性已经被覆盖，将会由nvwa接管启动过程，nvwa借助criu对现场进行恢复。恢复成功后，向systemd发送通知，修改服务的main pid。

## nvwa网络配置的保存和恢复(开发中)

当前nvwa主要关注以下的网络配置信息：

+ ip
+ route
+ iptables

这三类信息分别通过以下命令进行保存和恢复:

+ ip addr save / ip addr restore
+ ip route save / ip route restore
+ iptables-save / iptables-restore
