# Local Disk Manager (LDM)
<img width="150" align='right' alt="Hwameistor Logo" src="https://avatars.githubusercontent.com/u/98022243?s=200&v=4">

README: [English](https://github.com/hwameistor/local-disk-manager/blob/main/README.md) ｜ [中文](https://github.com/hwameistor/local-disk-manager/blob/main/README-zh.md)

`local-disk-manager (LDM)` is a sub module of [HwameiStor](https://github.com/hwameistor/helm-charts). 

Local Disk Manager (LDM) is one of modules of HwameiStor. LDM is to simplify the management of disks on nodes. It can abstract the disk on the node into a resource and can be monitored and managed. It's a daemon that will be deployed on each node, then detect the disk on the node, abstract it into local disk (LD) resources and save it to kubernetes.

At present, the LDM project is still in the alpha stage.

## Architecture of HwameiStor

![image](https://github.com/hwameistor/local-disk-manager/blob/main/doc/design/HwameiStor-arch.png)

## Concepts

**LocalDisk(LD)**: LDM abstracts disk resources into objects in k8s. A `LD` resource object represents the disk resources on the host.

**LocalDiskClaim(LDC)**: The way to use disk, users can add a description of the disk to select the disk to be used.
> At present, LDC supports the following disk description options:
> - NodeName
> - Capacity
> - DiskType(e.g. HDD/SSD)

## Usage
If you want to entirely deploy HwameiStor, please refer to [here](https://github.com/hwameistor/helm-charts). If you just want to deploy LDM separately, you can refer to the following installation steps.

## Install Local Disk Manager

### 1. Clone this repo to your machine:
```
# git clone https://github.com/hwameistor/local-disk-manager.git
```

### 2. Change to deploy directory:
```
# cd deploy
```

### 3. Deploy CRDs and run local-disk-manager

#### 3.1 Deploy needed CRDs 
```
# kubectl apply -f deploy/crds/
```

#### 3.2 Deploy operators
```
# kubectl apply -f deploy/operator.yaml
```

### 4. Get LocalDisk Infomation
```
# kubectl get localdisk
10-6-118-11-sda    10-6-118-11                             Unclaimed
10-6-118-11-sdb    10-6-118-11                             Unclaimed
``` 
Get locally discovered disk resource information. There are four columns of information. 
- Column **NAME** represents how this disk is displayed in the cluster resources. 
- Column **NODEMATCH** indicates which host this disk is on. 
- Column **CLAIM** indicates which localdiskclaim statement this disk is used by. 
- Column **PHASE** represents the current state of the disk.

`kuebctl get localdisk <name> -o yaml` View more information about disks.

### 5. Claim Available Disks

#### 5.1 Apply a LocalDiskClaim
Modify ***spec.nodeName*** field according to your actual situation before apply. 
```
# kubectl apply -f deploy/samples/resources/hwameistor.io_v1alpha1_localdiskclaim_cr.yaml
```

Allocate available disks by issuing a disk usage request. In the description of the request, you can add disk requirements, such as disk type and disk capacity

#### 5.2 Get LocalDiskClaim Infomation
```
# kubectl get localdiskclaim <name>
```
Check the status of claim. If there is a disk available, you will see that the status is bound and the status is on the corresponding localdisk is Claimed and points to the claim that references the disk.

## Use CSI Disk Volume
`CSI Disk Volume` provides raw disk based volume for workload. This function is **enabled by default.**

The following are examples of how to use two volume types ***Block and Filesystem***：

### Block Volume 
```shell
# kubectl apply -f deploy/ioping-disk.yaml
```
This deployment yaml deploys a ioping container that uses block volumes.

### Filesystem Volume
```shell
# kubectl apply -f deploy/ioping-fs.yaml
```
This deployment yaml deploys a ioping container that uses Filesystem volumes. 

## Roadmap

| Feature                   | Status | Release | TP Date | GA Date | Description                                                  |
| ------------------------- |--------|---------|---------| ------- | ------------------------------------------------------------ |
| CSI for disk volume       | Completed   |    | 2022.Q2 |         | CSI driver for provisioning Local Pvs with bare disk         |
| Disk management           | Planed |         |         |         | Disk management, disk allocation, disk event aware processing|
| Disk health management    | Planed |         |         |         | Fault prediction, status information reporting and so on     |
| HA disk Volume            | Planed |         |         |         | Disk Volume with HA                                          |                                      

## Feedbacks
Please submit any feedback and issue at: [Issues](https://github.com/hwameistor/local-disk-manager/issues)