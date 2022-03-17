# Local Disk Manager (LDM)
<img width="150" align='right' alt="Hwameistor Logo" src="https://avatars.githubusercontent.com/u/98022243?s=200&v=4">

README: [English](https://github.com/hwameistor/local-disk-manager/blob/main/README.md) ｜ [中文](https://github.com/hwameistor/local-disk-manager/blob/main/README-zh.md)

Local Disk Manager是HwameiStor系统的一个重要功能模块。`local-disk-manager (LDM)` 旨在简化管理节点上面的磁盘。它将磁盘抽象成一种可以被管理和监控的资源。它本身是一种 Daemonset 对象，集群中每一个节点都会运行该服务，通过该服务检测存在的磁盘并将其转换成相应的资源 LocalDisk。

目前 LDM 还处于 `alpha` 阶段。

## HwameiStor系统架构图

![image](https://github.com/hwameistor/local-disk-manager/blob/main/doc/design/HwameiStor-arch.png)


## 基本概念

**LocalDisk(LD)**: LDM 抽象的磁盘资源，一个 `LD ` 代表了节点上面的一块物理磁盘。

**LocalDiskClaim(LDC)**: 系统使用磁盘的方式，通过创建 `LDC` 对象来向系统申请磁盘。用户可以添加一些对磁盘的描述来选择磁盘。
> 当前阶段，LDC 支持以下对磁盘的描述选项：
> - NodeName
> - Capacity
> - DiskType(e.g. HDD/SSD)

## 用法
如果你想完整的部署 HwameiStor，请参考[这里]( https://github.com/hwameistor/helm-charts )。 如果你只是想单独部署 LDM，可以参考下面的步骤进行安装。

## 安装 Local Disk Manager

### 1. 克隆 repo 到本机:
```
# git clone https://github.com/hwameistor/local-disk-manager.git
```

### 2. 进入 repo 对应的目录
```
# cd deploy
```

### 3. 安装 CRDs 和 运行 LocalDiskManager

#### 3.1 安装 LocalDisk 和 LocalDiskClaim 的 CRD
```
# kubectl apply -f deploy/crds/
```

#### 3.2 安装权限认证的 CR 以及 LDM 的 Operators
```
# kubectl apply -f deploy/
```

### 4. 查看 LocalDisk 信息
```
# kubectl get localdisk
10-6-118-11-sda    10-6-118-11                             Unclaimed
10-6-118-11-sdb    10-6-118-11                             Unclaimed
```
该命令用于获取集群中磁盘资源信息，获取的信息总共有四列，含义分别如下：

- **NAME：** 代表磁盘在集群中的名称。
- **NODEMATCH：** 表明磁盘所在的节点名称。
- **CLAIM：** 表明这个磁盘是被哪个 `Claim` 所引用。
- **PHASE：** 表明这个磁盘当前的状态。

通过`kuebctl get localdisk <name> -o yaml` 查看更多关于某块磁盘的信息。

### 5. 申请可用磁盘

#### 5.1 创建 LocalDiskClaim
```
# kubectl apply -f deploy/samples/hwameistor.io_v1alpha1_localdiskclaim_cr.yaml
```
该命令用于创建一个磁盘使用的申请请求。在这个 yaml 文件里面，你可以在 description 字段添加对申请磁盘的描述，比如磁盘类型、磁盘的容量等等。

#### 5.2 查看 LocalDiskClaim 信息
```
# kubectl get localdiskclaim <name>
```
查看 `Claim` 的 Status 字段信息。如果存在可用的磁盘，你将会看到该字段的值为 `Bound`。

## 路线规划图

| Feature                | Status | Release | TP Date | GA Date | Description                                                  |
| :--------------------- | ------ | ------- | ------- | ------- | ------------------------------------------------------------ |
| CSI for disk volume    | Planed |         |         |         | `Disk` 模式下创建数据卷的 `CSI` 接口                            |
| Disk management        | Planed |         |         |         | 磁盘管理、磁盘分配、磁盘事件感知处理                              |
| Disk health management | Planed |         |         |         | 磁盘健康管理，包括故障预测和状态上报等等                           |
| HA disk Volume         | Planed |         |         |         | Disk 数据卷的高可用                                            |                                      

## 反馈
如果你有任何的疑问和建议，请反馈至 [Issues](https://github.com/hwameistor/local-disk-manager/issues)