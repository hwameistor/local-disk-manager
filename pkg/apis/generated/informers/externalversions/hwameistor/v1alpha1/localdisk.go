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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	versioned "github.com/hwameistor/local-disk-manager/pkg/apis/generated/clientset/versioned"
	internalinterfaces "github.com/hwameistor/local-disk-manager/pkg/apis/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/hwameistor/local-disk-manager/pkg/apis/generated/listers/hwameistor/v1alpha1"
	hwameistorv1alpha1 "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// LocalDiskInformer provides access to a shared informer and lister for
// LocalDisks.
type LocalDiskInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.LocalDiskLister
}

type localDiskInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewLocalDiskInformer constructs a new informer for LocalDisk type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewLocalDiskInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredLocalDiskInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredLocalDiskInformer constructs a new informer for LocalDisk type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredLocalDiskInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.HwameistorV1alpha1().LocalDisks().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.HwameistorV1alpha1().LocalDisks().Watch(context.TODO(), options)
			},
		},
		&hwameistorv1alpha1.LocalDisk{},
		resyncPeriod,
		indexers,
	)
}

func (f *localDiskInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredLocalDiskInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *localDiskInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&hwameistorv1alpha1.LocalDisk{}, f.defaultInformer)
}

func (f *localDiskInformer) Lister() v1alpha1.LocalDiskLister {
	return v1alpha1.NewLocalDiskLister(f.Informer().GetIndexer())
}