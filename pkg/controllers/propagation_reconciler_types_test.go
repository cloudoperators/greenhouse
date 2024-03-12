// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers_test

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/controllers"
	"github.com/cloudoperators/greenhouse/pkg/controllers/fixtures"
)

type ObjectStripper interface {
	StripObject(client.Object) (client.Object, error)
}

type TestDummyPropagationReconciler struct {
	controllers.PropagationReconciler
}

func (r *TestDummyPropagationReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.EmptyObj = &fixtures.Dummy{}
	r.EmptyObjList = &fixtures.DummyList{}
	r.CRDName = "dummies.greenhouse.sap"
	r.StripObjectWrapper = StripObject
	r.HandlerFunc = r.ListObjectsAsReconcileRequests

	return r.BaseSetupWithManager(name, mgr)
}

func (r *TestDummyPropagationReconciler) ListObjectsAsReconcileRequests(ctx context.Context, _ client.Object) []ctrl.Request {
	res := []ctrl.Request{}

	objList, ok := r.ListObjects(ctx).(*fixtures.DummyList)
	if !ok {
		return res
	}

	for _, obj := range objList.Items {
		res = append(res, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(obj.DeepCopy())})
	}

	return res
}

func StripObject(in client.Object) (client.Object, error) {
	tm, ok := in.(*fixtures.Dummy)
	if !ok {
		return nil, fmt.Errorf("error: %T is not a dummy", in)
	}

	typeMeta := metav1.TypeMeta{
		Kind:       in.GetObjectKind().GroupVersionKind().Kind,
		APIVersion: in.GetObjectKind().GroupVersionKind().GroupVersion().String(),
	}
	objectMeta := metav1.ObjectMeta{
		Name:        in.GetName(),
		Namespace:   in.GetNamespace(),
		Labels:      in.GetLabels(),
		Annotations: in.GetAnnotations(),
	}

	return &fixtures.Dummy{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       tm.Spec,
	}, nil
}
