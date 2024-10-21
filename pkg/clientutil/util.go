// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greehouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// IsSecretContainsKey checks whether the given secret contains a key.
func IsSecretContainsKey(s *corev1.Secret, key string) bool {
	if s.Data == nil {
		return false
	}
	v, ok := s.Data[key]
	return ok && v != nil
}

// GetSecretKeyFromSecretKeyReference returns the value of the secret identified by SecretKeyReference or an error.
func GetSecretKeyFromSecretKeyReference(ctx context.Context, c client.Client, namespace string, secretReference greenhousev1alpha1.SecretKeyReference) (string, error) {
	var secret = new(corev1.Secret)
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretReference.Name}, secret); err != nil {
		return "", err
	}
	if v, ok := secret.Data[secretReference.Key]; ok {
		// Trim newline characters from the end of the string.
		stringValue := string(v)
		stringValue = strings.TrimRight(stringValue, "\n")
		return stringValue, nil
	}
	return "", fmt.Errorf("secret %s/%s does not contain key %s", namespace, secretReference.Name, secretReference.Key)
}

// GetKubernetesVersion returns the kubernetes git version using the discovery client.
func GetKubernetesVersion(restClientGetter genericclioptions.RESTClientGetter, clientSet kubernetes.Interface) (*version.Info, error) {
	clientSet.Discovery().ServerVersion()
	dc, err := restClientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return dc.ServerVersion()
}

// Searches for a directory upwards starting from the given path.
func FindDirUpwards(path, dirName string, maxSteps int) (string, error) {
	return findRecursively(path, dirName, maxSteps, 0)
}

func findRecursively(path, dirName string, maxSteps, steps int) (string, error) {
	if path == "/" {
		return "", fmt.Errorf("root reached. directory not found: %s", dirName)
	}
	if maxSteps == steps {
		return "", fmt.Errorf("max steps reached. directory not found: %s", dirName)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	dirPath := filepath.Join(absPath, dirName)
	if _, err = os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			steps++
			return findRecursively(filepath.Join(absPath, ".."), dirName, maxSteps, steps)
		}
		return "", err
	}
	return dirPath, nil
}

// isMarkedForDeletion - checks if the cluster has deletion annotation and deletion schedule annotation
func isMarkedForDeletion(annotations map[string]string) bool {
	_, deletionMarked := annotations[greehouseapis.MarkClusterDeletionAnnotation]
	_, scheduleExists := annotations[greehouseapis.ScheduleClusterDeletionAnnotation]
	return deletionMarked && scheduleExists
}

// ExtractDeletionSchedule - extracts the deletion schedule from the annotation in time.DateTime format
func ExtractDeletionSchedule(annotations map[string]string) (bool, time.Time, error) {
	if annotations == nil {
		return false, time.Time{}, nil
	}
	_, deletionMarked := annotations[greehouseapis.MarkClusterDeletionAnnotation]
	deletionSchedule, scheduleExists := annotations[greehouseapis.ScheduleClusterDeletionAnnotation]
	if deletionMarked && scheduleExists {
		schedule, err := time.Parse(time.DateTime, deletionSchedule)
		return scheduleExists, schedule, err
	}
	return scheduleExists, time.Time{}, nil
}

// ShouldProceedDeletion - checks if the deletion should be allowed if the schedule has elapsed
func ShouldProceedDeletion(now, schedule time.Time) (bool, error) {
	// time.Before() compares two time objects
	// schedule is formatted as time.DateTime
	// so we need to format now as well to time.DateTime otherwise it will always return false
	formattedNow, err := ParseDateTime(now)
	if err != nil {
		return false, err
	}
	return !formattedNow.Before(schedule), nil
}

// FilterClustersBeingDeleted - filters out the clusters that are marked for deletion
func FilterClustersBeingDeleted(clusters *greenhousev1alpha1.ClusterList) *greenhousev1alpha1.ClusterList {
	// Iterate over the cluster list in reverse to safely remove elements to prevent index shifting when an item is removed.
	for i := len(clusters.Items) - 1; i >= 0; i-- {
		c := clusters.Items[i]
		if isMarkedForDeletion(clusters.Items[i].GetAnnotations()) || c.GetDeletionTimestamp() != nil {
			// Remove the cluster marked for deletion by slicing the array
			clusters.Items = append(clusters.Items[:i], clusters.Items[i+1:]...)
		}
	}
	return clusters
}

// ParseDateTime - parses the time object to time.DateTime format
func ParseDateTime(t time.Time) (time.Time, error) {
	layout := t.Format(time.DateTime)
	return time.Parse(time.DateTime, layout)
}
