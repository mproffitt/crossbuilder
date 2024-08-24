package main

import (
	"github.com/mproffitt/crossbuilder/examples/apis/v1alpha1"

	xgt "github.com/crossplane-contrib/function-go-templating/input/v1beta1"
	xpt "github.com/crossplane-contrib/function-patch-and-transform/input/v1beta1"
	xapiextv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type builder struct{}

var Builder = builder{}

func (b *builder) GetCompositeTypeRef() build.ObjectKindReference {
	return build.ObjectKindReference{
		GroupVersionKind: v1alpha1.XExampleGroupVersionKind,
		Object:           &xgt.GoTemplate{},
	}
}

func (b *builder) Build(c build.CompositionSkeleton) {
	c.WithName("pipelineexample").
		WithMode(xapiextv1.CompositionModePipeline).
		WithLabels(map[string]string{
			"example": "pipeline",
		})

	// Load the template
	var (
		template string
		err      error
	)
	template, err = build.LoadTemplate("compositions/pipelineexample/templates/*")
	if err != nil {
		panic(err)
	}

	c.NewPipelineStep("test-step").
		WithFunctionRef(xapiextv1.FunctionReference{
			Name: "function-go-templating",
		}).
		WithInput(build.ObjectKindReference{
			Object: &xgt.GoTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "gotemplating.fn.crossplane.io/v1beta1",
					Kind:       "GoTemplate",
				},
				Source: xgt.InlineSource,
				Inline: &xgt.TemplateSourceInline{
					Template: template,
				},
			},
		}).
		WithPatches(
			"resource-1",
			xpt.ComposedPatch{
				Type: xpt.PatchTypeFromCompositeFieldPath,
				Patch: xpt.Patch{
					FromFieldPath: strPtr("spec.containers[0].image"),
					ToFieldPath:   strPtr("spec.containers[0].image"),
				},
			},
			xpt.ComposedPatch{
				Type: xpt.PatchTypeFromCompositeFieldPath,
				Patch: xpt.Patch{
					FromFieldPath: strPtr("metadata.labels[app]"),
					ToFieldPath:   strPtr("metadata.labels[app]"),
				},
			},
		)
	c.NewPipelineStep("test-step-2").
		WithFunctionRef(xapiextv1.FunctionReference{
			Name: "function-patch-and-transform",
		}).
		WithInput(build.ObjectKindReference{
			Object: &xpt.Resources{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "pt.crossplane.io/v1beta1",
					Kind:       "Resources",
				},
				Resources: []xpt.ComposedTemplate{
					{
						Name: "resource-2",
						Base: &runtime.RawExtension{
							Object: &v1alpha1.XExample{},
						},
					},
				},
			},
		}).
		WithPatches(
			"resource-2",
			xpt.ComposedPatch{
				Type: xpt.PatchTypeFromCompositeFieldPath,
				Patch: xpt.Patch{
					FromFieldPath: strPtr("status.atProvider.something"),
					ToFieldPath:   strPtr("spec.forProvider.something"),
				},
			},
			xpt.ComposedPatch{
				Type: xpt.PatchTypeFromCompositeFieldPath,
				Patch: xpt.Patch{
					FromFieldPath: strPtr("metadata.labels[app]"),
					ToFieldPath:   strPtr("spec.forProvider.labels[app]"),
				},
			},
		)
}

func strPtr(s string) *string {
	return &s
}
