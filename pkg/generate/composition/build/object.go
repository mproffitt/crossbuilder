package build

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Object is an extension of the k8s runtime.Object with additional functions
// that are required by Crossbuildec.
type Object interface {
	runtime.Object
	SetGroupVersionKind(gvk schema.GroupVersionKind)
}

// ObjectKindReference contains the group version kind and instance of a
// runtime.Object.
type ObjectKindReference struct {
	// GroupVersionKind is the GroupVersionKind for the composite type.
	GroupVersionKind schema.GroupVersionKind

	// Object is an instance of the composite type.
	Object Object
}

func makeLabelPaths(keys []string) []string {
	paths := make([]string, len(keys))
	for i, k := range keys {
		paths[i] = fmt.Sprintf("metadata.labels[%s]", k)
	}
	return paths
}

func makeAnnotationPaths(keys []string) []string {
	paths := make([]string, len(keys))
	for i, k := range keys {
		paths[i] = fmt.Sprintf("metadata.annotations[%s]", k)
	}
	return paths
}
