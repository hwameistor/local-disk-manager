//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2022 Contributors to The HwameiStor project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplaceDisk) DeepCopyInto(out *ReplaceDisk) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplaceDisk.
func (in *ReplaceDisk) DeepCopy() *ReplaceDisk {
	if in == nil {
		return nil
	}
	out := new(ReplaceDisk)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplaceDisk) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplaceDiskList) DeepCopyInto(out *ReplaceDiskList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ReplaceDisk, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplaceDiskList.
func (in *ReplaceDiskList) DeepCopy() *ReplaceDiskList {
	if in == nil {
		return nil
	}
	out := new(ReplaceDiskList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplaceDiskList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplaceDiskSpec) DeepCopyInto(out *ReplaceDiskSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplaceDiskSpec.
func (in *ReplaceDiskSpec) DeepCopy() *ReplaceDiskSpec {
	if in == nil {
		return nil
	}
	out := new(ReplaceDiskSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplaceDiskStatus) DeepCopyInto(out *ReplaceDiskStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplaceDiskStatus.
func (in *ReplaceDiskStatus) DeepCopy() *ReplaceDiskStatus {
	if in == nil {
		return nil
	}
	out := new(ReplaceDiskStatus)
	in.DeepCopyInto(out)
	return out
}