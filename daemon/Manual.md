# FrozenGo Daemon 安装手册
## 安装依赖 (Daemon在安装时会自动尝试安装，若仍然提示限制无效，请手动安装该工具)
1.cgroup<br/>
debian/ubuntu/other:
```bash
apt-get -y install cgroup-bin
```
RedHat(CentOS,FedoraServer,RHEL):
```bash
yum -y install libcgroup
```
## 安装步骤
```bash
wget http://frozen-go.oss-cn-hangzhou.aliyuncs.com/bin.tar.gz
tar -xvzf bin.tar.gz
mkdir FrozenGo
mv ./bin ./FrozenGo/
cd FrozenGo/bin
./frozen
```
## 后记
请注意，目前限制测试兼容CentOS 6.x 系列，Ubuntu 16.04 网卡可能出现Invalid arguments...<br />
Mount技术的磁盘限制还不成熟，可以用但是千万不要乱删服<br />
给出的提示一定要认真看！<br />
当前Daemon可运行的系统：Debian9.1，UbuntuServer17.04，FedoraServer26，deepin15.4.1，CentOS6.X<br/>

