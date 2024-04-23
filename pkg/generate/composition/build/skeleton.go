package build

import (
	xpt "github.com/crossplane-contrib/function-patch-and-transform/input/v1beta1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	xapiextv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
)

const (
	errEmptyCompositionname                 = "composition name must not be empty"
	errFmtBuildComposedTemplate             = "cannot build composed template at index %d"
	errFmtInvalidPatch                      = "invalid patch at index %d"
	errPatchFromFieldPath                   = "fromFieldPath is invalid"
	errPatchToFieldPath                     = "toFieldPath is invalid"
	errPatchRequireField                    = "missing field %s"
	errPatchCombineEmptyVariables           = "no variables given"
	errFmtPatchCombineVariableFromFieldPath = "fromFieldPath of variable at index %d is invalid"
	errUnknownPatchType                     = "unknown patch type %s"
	errParseRegisteredCompositePaths        = "cannot parse registered composite paths"
	errParseRegisteredComposedPaths         = "cannot parse registered composed paths"
	errInvalidPipelineMode                  = "invalid mode for pipeline composition"
	errInvalidResourcesMode                 = "invalid mode for resources composition"
	errFmtSetupComposition                  = "failed to setup composition"
	errFmtInvalidPatchAndTransform          = "invalid patch-and-transform function ref"
	errNilObject                            = "object must not be nil"

	labelKeyClaimName      = "crossplane.io/claim-name"
	labelKeyClaimNamespace = "crossplane.io/claim-namespace"
)

var (
	// KnownCompositeAnnotations are annotations that will be registered by
	// default
	KnownCompositeAnnotations = []string{}

	// KnownCompositeLabels are labels that will be registered by default.
	KnownCompositeLabels = []string{
		labelKeyClaimName,
		labelKeyClaimNamespace,
	}
	// KnownResourceAnnotations are annotations that will be registered by
	// default
	KnownResourceAnnotations = []string{
		meta.AnnotationKeyExternalName,
		meta.AnnotationKeyExternalCreatePending,
		meta.AnnotationKeyExternalCreateSucceeded,
		meta.AnnotationKeyExternalCreateFailed,
	}
	// KnownResourceLabels are labels that will be registered by default.
	KnownResourceLabels = []string{}
)

// ComposedTemplateSkeleton represents the draft for a compositionSkeleton composeTemplateSkeleton.
type ComposedTemplateSkeleton interface {
	// WithName sets the name of this composeTemplateSkeleton.
	WithName(name string) ComposedTemplateSkeleton

	// WithPatches adds the following patches to this composeTemplateSkeleton.
	WithPatches(patches ...xapiextv1.Patch) ComposedTemplateSkeleton

	// WithUnsafePatches is similar to WithPatches but the field paths of the
	// composeTemplateSkeletons will not be validated.
	WithUnsafePatches(patches ...xapiextv1.Patch) ComposedTemplateSkeleton

	// WithConnectionDetails adds the following connection details to this
	// composeTemplateSkeleton.
	WithConnectionDetails(connectionDetails ...xapiextv1.ConnectionDetail) ComposedTemplateSkeleton

	// WithReadinessChecks adds the following readiness checks to this
	// composeTemplateSkeleton.
	WithReadinessChecks(checks ...xapiextv1.ReadinessCheck) ComposedTemplateSkeleton

	// RegisterAnnotations marks the given resource annotations as safe
	// so they will be treated as a valid field in patch paths.
	RegisterAnnotations(annotationKeys ...string) ComposedTemplateSkeleton

	// RegisterLabels marks the given resource label as safe
	// so they will be treated as a valid field in patch paths.
	RegisterLabels(labelsKeys ...string) ComposedTemplateSkeleton

	// RegisterFieldPaths marks the given resource paths as safe so ti will
	// be treated a valid in patch paths.
	RegisterFieldPaths(paths ...string) ComposedTemplateSkeleton
}

// CompositionSkeleton represents the build time state of a composition.
type CompositionSkeleton interface {
	// WithName sets the metadata.name of the composition to be built.
	WithName(name string) CompositionSkeleton

	// WithLabels sets the metadata.labels of the composition to be built.
	WithLabels(labels map[string]string) CompositionSkeleton

	// WithAnnotations sets the metadata.annotations of the composition to be built.
	WithAnnotations(annotations map[string]string) CompositionSkeleton

	// WithMode sets the mode of this compositionSkeleton.
	WithMode(mode xapiextv1.CompositionMode) CompositionSkeleton

	// NewResource creates a new ComposedTemplateSkeleton with the given base.
	NewResource(base ObjectKindReference) ComposedTemplateSkeleton

	// WithPipeline adds the following pipeline to this compositionSkeleton.
	NewPipelineStep(step string) PipelineStepSkeleton

	// WithPublishConnectionDetailsWithStoreConfig sets the
	// PublishConnectionDetailsWithStoreConfig of this CompositionSkeleton.
	WithPublishConnectionDetailsWithStoreConfig(ref *xapiextv1.StoreConfigReference) CompositionSkeleton

	// WithWriteConnectionSecretsToNamespace sets the
	// WriteConnectionSecretsToNamespace of this compositionSkeleton.
	WithWriteConnectionSecretsToNamespace(namespace *string) CompositionSkeleton

	// RegisterCompositeAnnotations marks the given composite annotations as safe
	// so it will be treated as a valid field in patch paths.
	RegisterCompositeAnnotations(annotationKeys ...string) CompositionSkeleton

	// RegisterCompositeLabels marks the given composite labels as safe
	// so it will be treated as a valid field in patch paths.
	RegisterCompositeLabels(labelKeys ...string) CompositionSkeleton

	// RegisterCompositeFieldPaths marks the given composite paths as safe so
	// they will be treated a valid in patch paths.
	RegisterCompositeFieldPaths(paths ...string) CompositionSkeleton
}

// PipelineStepSkeleton represents the build time state of a pipeline step.
type PipelineStepSkeleton interface {

	// WithSteps adds the following steps to this pipeline.
	WithFunctionRef(ref xapiextv1.FunctionReference) PipelineStepSkeleton

	// WithInput sets the input of this pipeline step.
	WithInput(input ObjectKindReference) PipelineStepSkeleton

	// WithPatches adds the following patches to this pipeline step.
	//
	// Will automatically register the `patch-and-transform` function if not
	// already registered.
	WithPatches(name string, patches ...xpt.ComposedPatch) PipelineStepSkeleton

	// WithPatch adds the following patch to this pipeline step.
	WithPatch(name string, patch xpt.ComposedPatch) PipelineStepSkeleton
}
