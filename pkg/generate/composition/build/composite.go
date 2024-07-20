package build

import (
	"fmt"

	xpt "github.com/crossplane-contrib/function-patch-and-transform/input/v1beta1"
	xapiextv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	functionPatchAndTransform = "function-patch-and-transform"
)

type compositionSkeleton struct {
	composite                               ObjectKindReference
	labels                                  map[string]string
	annotations                             map[string]string
	registeredPaths                         []string
	name                                    string
	mode                                    xapiextv1.CompositionMode
	composeTemplateSkeletons                []*composeTemplateSkeleton
	pipeline                                []*pipelineStepSkeleton
	publishConnectionDetailsWithStoreConfig *xapiextv1.StoreConfigReference
	writeConnectionSecretsToNamespace       *string
}

// RegisterCompositeAnnotations marks the given composite annotations as safe so
// it will be treated as a valid field in patch paths.
func (c *compositionSkeleton) RegisterCompositeAnnotations(annotionKeys ...string) CompositionSkeleton {
	paths := make([]string, len(annotionKeys))
	for i, k := range annotionKeys {
		paths[i] = fmt.Sprintf("metadata.annotations[%s]", k)
	}
	return c.RegisterCompositeFieldPaths(paths...)
}

// RegisterCompositeLabels marks the given composite labels as safe so it
// will be treated as a valid field in patch paths.
func (c *compositionSkeleton) RegisterCompositeLabels(labelKeys ...string) CompositionSkeleton {
	paths := make([]string, len(labelKeys))
	for i, k := range labelKeys {
		paths[i] = fmt.Sprintf("metadata.labels[%s]", k)
	}
	return c.RegisterCompositeFieldPaths(paths...)
}

// RegisterCompositeFieldPaths marks the given composite paths as safe so ti will
// be treated a valid in patch paths.
func (c *compositionSkeleton) RegisterCompositeFieldPaths(path ...string) CompositionSkeleton {
	c.registeredPaths = append(c.registeredPaths, path...)
	return c
}

// WithName sets the metadata.name of the composition to be built.
func (c *compositionSkeleton) WithName(name string) CompositionSkeleton {
	c.name = name
	return c
}

// WithLabels sets the metadata.labels of the composition to be built.
func (c *compositionSkeleton) WithLabels(labels map[string]string) CompositionSkeleton {
	if c.labels == nil {
		c.labels = make(map[string]string)
	}

	for k, v := range labels {
		c.labels[k] = v
	}
	return c
}

// WithAnnotations sets the metadata.annotations of the composition to be built.
func (c *compositionSkeleton) WithAnnotations(annotations map[string]string) CompositionSkeleton {
	if c.annotations == nil {
		c.annotations = make(map[string]string)
	}

	for k, v := range annotations {
		c.annotations[k] = v
	}
	return c
}

// WithMode sets the mode of this compositionSkeleton.
func (c *compositionSkeleton) WithMode(mode xapiextv1.CompositionMode) CompositionSkeleton {
	c.mode = mode
	return c
}

// NewResource creates a new composeTemplateSkeleton with the given base.
func (c *compositionSkeleton) NewResource(base ObjectKindReference) ComposedTemplateSkeleton {
	res := &composeTemplateSkeleton{
		base:                base,
		compositionSkeleton: c,
	}
	c.composeTemplateSkeletons = append(c.composeTemplateSkeletons, res)
	return res
}

func (c *compositionSkeleton) NewPipelineStep(step string) PipelineStepSkeleton {
	ps := &pipelineStepSkeleton{
		step:                step,
		compositionSkeleton: c,
	}

	c.pipeline = append(c.pipeline, ps)
	return ps
}

// WithPublishConnectionDetailsWithStoreConfig sets the
// PublishConnectionDetailsWithStoreConfig of this CompositionSkeleton.
func (c *compositionSkeleton) WithPublishConnectionDetailsWithStoreConfig(ref *xapiextv1.StoreConfigReference) CompositionSkeleton {
	c.publishConnectionDetailsWithStoreConfig = ref
	return c
}

// WithWriteConnectionSecretsToNamespace sets the
// WriteConnectionSecretsToNamespace of this compositionSkeleton.
func (c *compositionSkeleton) WithWriteConnectionSecretsToNamespace(namespace *string) CompositionSkeleton {
	c.writeConnectionSecretsToNamespace = namespace
	return c
}

// ToComposition generates a Crossplane compositionSkeleton from this compositionSkeleton.
func (c *compositionSkeleton) ToComposition() (xapiextv1.Composition, error) {
	if c.name == "" {
		return xapiextv1.Composition{}, errors.New(errEmptyCompositionname)
	}

	c.RegisterCompositeAnnotations(KnownCompositeAnnotations...)
	c.RegisterCompositeLabels(KnownCompositeLabels...)

	var (
		composedTemplates []xapiextv1.ComposedTemplate
		pipelineSteps     []xapiextv1.PipelineStep
		err               error
	)

	switch c.mode {
	case "", xapiextv1.CompositionModeResources:
		c.mode = xapiextv1.CompositionModeResources
		composedTemplates, err = c.setupComposed()
	case xapiextv1.CompositionModePipeline:
		pipelineSteps, err = c.setupPipeline()
	}

	if err != nil {
		return xapiextv1.Composition{}, errors.Wrap(err, errFmtSetupComposition)
	}

	comp := xapiextv1.Composition{
		Spec: xapiextv1.CompositionSpec{
			CompositeTypeRef:                  xapiextv1.TypeReferenceTo(c.composite.GroupVersionKind),
			Mode:                              &c.mode,
			Resources:                         composedTemplates,
			Pipeline:                          pipelineSteps,
			WriteConnectionSecretsToNamespace: c.writeConnectionSecretsToNamespace,
			PublishConnectionDetailsWithStoreConfigRef: c.publishConnectionDetailsWithStoreConfig,
		},
	}

	comp.SetGroupVersionKind(xapiextv1.CompositionGroupVersionKind)
	comp.SetName(c.name)
	comp.SetLabels(c.labels)
	comp.SetAnnotations(c.annotations)
	comp.SetCreationTimestamp(metav1.Time{})
	return comp, nil
}

func (c *compositionSkeleton) setupComposed() ([]xapiextv1.ComposedTemplate, error) {
	if c.mode != xapiextv1.CompositionModeResources {
		return nil, errors.New(errInvalidResourcesMode)
	}

	composedTemplates := make([]xapiextv1.ComposedTemplate, len(c.composeTemplateSkeletons))
	for i, c := range c.composeTemplateSkeletons {
		ct, err := c.ToComposedTemplate()
		if err != nil {
			return nil, errors.Wrapf(err, errFmtBuildComposedTemplate, i)
		}
		composedTemplates[i] = ct
	}
	return composedTemplates, nil
}

func (c *compositionSkeleton) setupPipeline() ([]xapiextv1.PipelineStep, error) {
	if c.mode != xapiextv1.CompositionModePipeline {
		return nil, errors.New(errInvalidPipelineMode)
	}

	/*var pt *pipelineStepSkeleton
	{
		for _, v := range c.pipeline {
			if v.functionRef != nil && v.functionRef.Name == functionPatchAndTransform {
				pt = v
				break
			}
		}
	}

	if pt == nil {
		pt = &pipelineStepSkeleton{
			step:                "patch-and-transform",
			compositionSkeleton: c,
			functionRef: &xapiextv1.FunctionReference{
				Name: functionPatchAndTransform,
			},
			input: &ObjectKindReference{
				Object: &xpt.Resources{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "pt.crossplane.io/v1beta1",
						Kind:       "Resources",
					},
					Resources: make([]xpt.ComposedTemplate, 0),
				},
			},
		}
		c.pipeline = append(c.pipeline, pt)
	}

	oir := pt.input.Object.(*xpt.Resources)*/

	pipelineSteps := make([]xapiextv1.PipelineStep, len(c.pipeline))
	for i, p := range c.pipeline {
		pipelineSteps[i] = toPipelineStep(p)
		fmt.Printf("step: %+v\n", p.step)
		/*if p.patches == nil {
			continue
		}
		mapInputResource(oir, p.patches)*/
	}
	return pipelineSteps, nil
}

func mapInputResource(oir *xpt.Resources, patches map[string][]xpt.ComposedPatch) {
	var added bool = false

	for resourceName, resourcePatches := range patches {
		for i, oirp := range oir.Resources {
			if oirp.Name == resourceName {
				oir.Resources[i].Patches = append(oir.Resources[i].Patches, resourcePatches...)
				added = true
				break
			}
		}

		if !added {
			oir.Resources = append(oir.Resources, xpt.ComposedTemplate{
				Name:    resourceName,
				Patches: patches[resourceName],
			})
		}
	}
}

func toPipelineStep(p *pipelineStepSkeleton) xapiextv1.PipelineStep {
	var object xapiextv1.PipelineStep = xapiextv1.PipelineStep{
		Step:        p.step,
		FunctionRef: *p.functionRef,
	}

	if p.input != nil {
		object.Input = &runtime.RawExtension{
			Object: p.input.Object,
		}
	}
	return object
}
