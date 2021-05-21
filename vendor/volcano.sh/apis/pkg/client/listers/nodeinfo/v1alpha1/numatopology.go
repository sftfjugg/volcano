/*
Copyright 2021 The Volcano Authors.

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
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "volcano.sh/apis/pkg/apis/nodeinfo/v1alpha1"
)

// NumatopologyLister helps list Numatopologies.
// All objects returned here must be treated as read-only.
type NumatopologyLister interface {
	// List lists all Numatopologies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Numatopology, err error)
	// Numatopologies returns an object that can list and get Numatopologies.
	Numatopologies(namespace string) NumatopologyNamespaceLister
	NumatopologyListerExpansion
}

// numatopologyLister implements the NumatopologyLister interface.
type numatopologyLister struct {
	indexer cache.Indexer
}

// NewNumatopologyLister returns a new NumatopologyLister.
func NewNumatopologyLister(indexer cache.Indexer) NumatopologyLister {
	return &numatopologyLister{indexer: indexer}
}

// List lists all Numatopologies in the indexer.
func (s *numatopologyLister) List(selector labels.Selector) (ret []*v1alpha1.Numatopology, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Numatopology))
	})
	return ret, err
}

// Numatopologies returns an object that can list and get Numatopologies.
func (s *numatopologyLister) Numatopologies(namespace string) NumatopologyNamespaceLister {
	return numatopologyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// NumatopologyNamespaceLister helps list and get Numatopologies.
// All objects returned here must be treated as read-only.
type NumatopologyNamespaceLister interface {
	// List lists all Numatopologies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Numatopology, err error)
	// Get retrieves the Numatopology from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Numatopology, error)
	NumatopologyNamespaceListerExpansion
}

// numatopologyNamespaceLister implements the NumatopologyNamespaceLister
// interface.
type numatopologyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Numatopologies in the indexer for a given namespace.
func (s numatopologyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Numatopology, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Numatopology))
	})
	return ret, err
}

// Get retrieves the Numatopology from the indexer for a given namespace and name.
func (s numatopologyNamespaceLister) Get(name string) (*v1alpha1.Numatopology, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("numatopology"), name)
	}
	return obj.(*v1alpha1.Numatopology), nil
}
