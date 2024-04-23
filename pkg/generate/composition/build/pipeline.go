package build

import (
	xpt "github.com/crossplane-contrib/function-patch-and-transform/input/v1beta1"
	xapiextv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
)

type pipelineStepSkeleton struct {
	compositionSkeleton *compositionSkeleton
	step                string
	functionRef         *xapiextv1.FunctionReference
	input               *ObjectKindReference
	patches             map[string][]xpt.ComposedPatch
}

// WithFunctionRef sets the function reference for this pipeline step.
func (p *pipelineStepSkeleton) WithFunctionRef(ref xapiextv1.FunctionReference) PipelineStepSkeleton {
	p.functionRef = &ref
	return p
}

// WithInput sets the input for this pipeline step.
func (p *pipelineStepSkeleton) WithInput(input ObjectKindReference) PipelineStepSkeleton {
	p.input = &input
	return p
}

// WithStep sets the name for this pipeline step.
func (p *pipelineStepSkeleton) WithStep(step string) PipelineStepSkeleton {
	p.step = step
	return p
}

// WithPatches adds the following patches to this pipeline step.
func (p *pipelineStepSkeleton) WithPatches(name string, patches ...xpt.ComposedPatch) PipelineStepSkeleton {
	for _, patch := range patches {
		p.WithPatch(name, patch)
	}
	return p
}

// WithPatch adds the following patch to this pipeline step.
func (p *pipelineStepSkeleton) WithPatch(name string, patch xpt.ComposedPatch) PipelineStepSkeleton {
	if p.patches == nil {
		p.patches = make(map[string][]xpt.ComposedPatch)
	}

	if _, ok := p.patches[name]; !ok {
		p.patches[name] = make([]xpt.ComposedPatch, 0)
	}

	p.patches[name] = append(p.patches[name], patch)
	return p
}
