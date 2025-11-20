// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"encoding/json"
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// AppliedPropagatorAnnotation stores the list of last applied label keys on the destination object
	AppliedPropagatorAnnotation = "greenhouse.sap/last-applied-propagator"

	// PropagateLabelsAnnotation defines which label keys should be propagated from source to destination
	PropagateLabelsAnnotation = "greenhouse.sap/propagate-labels"
	// PropagateAnnotationsAnnotation defines which annotation keys should be propagated from source to destination
	PropagateAnnotationsAnnotation = "greenhouse.sap/propagate-annotations"
)

// appliedPropagatorState holds the state of previously applied label keys for cleanup purposes
// and is stored as JSON in the destination object's annotations.
type appliedPropagatorState struct {
	LabelKeys      []string `json:"labelKeys,omitempty"`
	AnnotationKeys []string `json:"annotationKeys,omitempty"`
}

// Propagator encapsulates a source and destination object between which
// label keys are propagated according to configured annotations.
type Propagator struct {
	src client.Object
	dst client.Object
}

// NewPropagator creates a new Propagator instance for syncing labels
// between a source and destination Kubernetes object.
func NewPropagator(src, dst client.Object) *Propagator {
	return &Propagator{
		src: src,
		dst: dst,
	}
}

// Apply - performs idempotent propagation of declared label and annotation keys based on
// the propagate-labels and propagate-annotations annotations on the source object. It adds or
// updates only the specified keys from src to dst, and removes any previously propagated keys
// that were removed in src or are no longer declared.
func (p *Propagator) Apply() client.Object {
	labelKeys := p.labelKeysToPropagate()
	annotationKeys := p.annotationKeysToPropagate()
	if len(labelKeys) == 0 && len(annotationKeys) == 0 {
		return p.cleanupTarget()
	}

	srcLabels := p.src.GetLabels()
	if srcLabels == nil {
		srcLabels = map[string]string{}
	}
	srcAnnotations := p.src.GetAnnotations()
	if srcAnnotations == nil {
		srcAnnotations = map[string]string{}
	}
	dstLabels := p.dst.GetLabels()
	if dstLabels == nil {
		dstLabels = map[string]string{}
	}
	dstAnnotations := p.dst.GetAnnotations()
	if dstAnnotations == nil {
		dstAnnotations = map[string]string{}
	}

	hasAny := false
	if len(labelKeys) > 0 && p.containsLabelToPropagate(labelKeys, srcLabels) {
		hasAny = true
	}
	if !hasAny && len(annotationKeys) > 0 && p.containsAnnotationToPropagate(annotationKeys, srcAnnotations) {
		hasAny = true
	}
	if !hasAny {
		return p.cleanupTarget()
	}

	appliedLabelKeys := p.syncTargetLabels(labelKeys, srcLabels, dstLabels)
	appliedAnnotationKeys := p.syncTargetAnnotations(annotationKeys, srcAnnotations, dstAnnotations)
	p.dst.SetLabels(dstLabels)
	p.dst.SetAnnotations(dstAnnotations)
	if len(appliedLabelKeys) > 0 || len(appliedAnnotationKeys) > 0 {
		p.storeAppliedState(appliedPropagatorState{LabelKeys: appliedLabelKeys, AnnotationKeys: appliedAnnotationKeys})
	} else {
		p.removeAppliedState()
	}

	return p.dst
}

// labelKeysToPropagate - retrieves the list of label keys from the propagate-labels annotation
// in the source object. Returns nil if missing, invalid, or empty.
func (p *Propagator) labelKeysToPropagate() []string {
	var keys []string
	annotation := strings.TrimSpace(p.src.GetAnnotations()[PropagateLabelsAnnotation])
	if annotation == "" {
		return nil
	}
	rawKeys := strings.Split(annotation, ",")
	for _, k := range rawKeys {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// annotationKeysToPropagate - retrieves the list of annotation keys from the propagate-annotations annotation
// in the source object. Returns nil if missing, invalid, or empty.
func (p *Propagator) annotationKeysToPropagate() []string {
	var keys []string
	annotation := strings.TrimSpace(p.src.GetAnnotations()[PropagateAnnotationsAnnotation])
	if annotation == "" {
		return nil
	}
	rawKeys := strings.Split(annotation, ",")
	for _, k := range rawKeys {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// containsLabelToPropagate - returns true if the source object contains
// at least one of the label keys declared for propagation.
func (p *Propagator) containsLabelToPropagate(keys []string, srcLabels map[string]string) bool {
	for _, k := range keys {
		if _, ok := srcLabels[k]; ok {
			return true
		}
	}
	return false
}

// containsAnnotationToPropagate - returns true if the source object contains
// at least one of the annotation keys declared for propagation.
func (p *Propagator) containsAnnotationToPropagate(keys []string, srcAnnotations map[string]string) bool {
	for _, k := range keys {
		if _, ok := srcAnnotations[k]; ok {
			return true
		}
	}
	return false
}

// syncTargetLabels - synchronizes label keys from src to dst and removes any previously applied keys
// that are no longer present. Returns the current list of successfully propagated keys.
func (p *Propagator) syncTargetLabels(keys []string, srcLabels, dstLabels map[string]string) []string {
	var appliedNow []string
	state := p.getAppliedState()
	for _, k := range keys {
		if v, ok := srcLabels[k]; ok {
			dstLabels[k] = v
			appliedNow = append(appliedNow, k)
		}
	}
	for _, k := range state.LabelKeys {
		if !slices.Contains(keys, k) || srcLabels[k] == "" {
			delete(dstLabels, k)
		}
	}
	return appliedNow
}

// syncTargetAnnotations - synchronizes annotation keys from src to dst and removes any previously applied keys
// that are no longer present. Returns the current list of successfully propagated annotation keys.
func (p *Propagator) syncTargetAnnotations(keys []string, srcAnn, dstAnn map[string]string) []string {
	var appliedNow []string
	state := p.getAppliedState()
	for _, k := range keys {
		if v, ok := srcAnn[k]; ok {
			dstAnn[k] = v
			appliedNow = append(appliedNow, k)
		}
	}
	for _, k := range state.AnnotationKeys {
		if !slices.Contains(keys, k) || srcAnn[k] == "" {
			delete(dstAnn, k)
		}
	}
	return appliedNow
}

// getAppliedState - reads the last-applied-propagator annotation from the destination object and unmarshal the
// previously applied label keys for later cleanup.
func (p *Propagator) getAppliedState() appliedPropagatorState {
	annotations := p.dst.GetAnnotations()
	if annotations == nil {
		return appliedPropagatorState{}
	}
	raw := annotations[AppliedPropagatorAnnotation]
	if raw == "" {
		return appliedPropagatorState{}
	}
	var state appliedPropagatorState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return appliedPropagatorState{}
	}
	return state
}

// storeAppliedState - saves the given appliedPropagatorState into the destination object's annotations.
func (p *Propagator) storeAppliedState(state appliedPropagatorState) {
	data, _ := json.Marshal(state)
	ann := p.dst.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	ann[AppliedPropagatorAnnotation] = string(data)
	p.dst.SetAnnotations(ann)
}

// removeAppliedState - deletes the applied propagator tracking annotation from the destination object.
func (p *Propagator) removeAppliedState() {
	ann := p.dst.GetAnnotations()
	if ann == nil {
		return
	}
	delete(ann, AppliedPropagatorAnnotation)
	p.dst.SetAnnotations(ann)
}

// cleanupTarget - removes any previously applied propagated labels from the destination object
// and deletes the tracking annotation. It is called when no label propagation should occur.
func (p *Propagator) cleanupTarget() client.Object {
	labels := p.dst.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	ann := p.dst.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	state := p.getAppliedState()
	for _, k := range state.LabelKeys {
		delete(labels, k)
	}
	for _, k := range state.AnnotationKeys {
		delete(ann, k)
	}
	p.dst.SetLabels(labels)
	p.dst.SetAnnotations(ann)
	p.removeAppliedState()
	return p.dst
}
