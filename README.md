# local-disk-manager

README: [English](https://github.com/hwameistor/local-disk-manager/blob/main/README.md) ｜ [中文](https://github.com/hwameistor/local-disk-manager/blob/main/README-zh.md)

`local-disk-manager` (LDM) is to simplify the management of disks on nodes. It can abstract the disk on the node into a resource and can be monitored and managed. It's a daemon that will be deployed on each node, then detect the disk on the node, abstract it into local disk (LD) resources and save it to kubernetes.

At present, the `LDM` project is still in the alpha stage.

## Concepts

**LocalDisk**: LDM abstracts disk resources into objects in k8s. A LocalDisk (LD) resource object represents the disk resources on the host.

**LocalDiskClaim**: The way to use disk, users can add a description of the disk to select the disk to be used.

## Usage

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

#### 3.1 Deploy LD and LDC CRDs
```
# kubectl apply -f deploy/crds/
```

#### 3.2 Deploy RBAC CRs and operators
```
# kubectl apply -f deploy/
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
```
# kubectl apply -f deploy/samples/hwameistor.io_v1alpha1_localdiskclaim_cr.yaml
```
Allocate available disks by issuing a disk usage request. In the description of the request, you can add disk requirements, such as disk type and disk capacity

#### 5.2 Get LocalDiskClaim Infomation
```
# kubectl get localdiskclaim <name>
```
Check the status of claim. If there is a disk available, you will see that the status is bound and the status is on the corresponding localdisk is Claimed and points to the claim that references the disk.

## Roadmap

| Feature                   | Status | Release | TP Date | GA Date | Description                                                  |
| ------------------------- | ------ | ------- | ------- | ------- | ------------------------------------------------------------ |
| CSI for disk volume       | Planed |         |         |         | Container Storage Interface                                  |
| Disk management           | Planed |         |         |         | Disk management, including various events and how the system should handle them when they occur |
| Disk health management    | Planed |         |         |         | Fault prediction, status information reporting and so on     |
