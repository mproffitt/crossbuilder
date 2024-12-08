package build

import (
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	xapiextv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

type composeTemplateSkeleton struct {
	compositionSkeleton *compositionSkeleton
	registeredPaths     []string
	name                *string
	base                ObjectKindReference
	patches             []patchSkeleton
	connectionDetails   []xapiextv1.ConnectionDetail
	readinessChecks     []xapiextv1.ReadinessCheck
}

// RegisterAnnotations marks the given resource annotations as safe
// so they will be treated as a valid field in patch paths.
func (c *composeTemplateSkeleton) RegisterAnnotations(annotionKeys ...string) ComposedTemplateSkeleton {
	return c.RegisterFieldPaths(makeAnnotationPaths(annotionKeys)...)
}

// RegisterLabels marks the given resource labels as safe
// so they will be treated as a valid field in patch paths.
func (c *composeTemplateSkeleton) RegisterLabels(labelKeys ...string) ComposedTemplateSkeleton {
	return c.RegisterFieldPaths(makeLabelPaths(labelKeys)...)
}

// RegisterFieldPaths marks the given resource paths as safe so they will
// be treated a valid in patch paths.
func (c *composeTemplateSkeleton) RegisterFieldPaths(paths ...string) ComposedTemplateSkeleton {
	c.registeredPaths = append(c.registeredPaths, paths...)
	return c
}

// WithName sets the name of this composeTemplateSkeleton.
func (c *composeTemplateSkeleton) WithName(name string) ComposedTemplateSkeleton {
	c.name = &name
	return c
}

// WithPatches adds the following patches to this composeTemplateSkeleton.
func (c *composeTemplateSkeleton) WithPatches(patches ...xapiextv1.Patch) ComposedTemplateSkeleton {
	for _, patch := range patches {
		c.patches = append(c.patches, patchSkeleton{
			patch:  patch,
			unsafe: false,
		})
	}
	return c
}

// WithUnsafePatches is similar to WithPatches but the field paths of the
// composeTemplateSkeletons will not be validated.
func (c *composeTemplateSkeleton) WithUnsafePatches(patches ...xapiextv1.Patch) ComposedTemplateSkeleton {
	for _, patch := range patches {
		c.patches = append(c.patches, patchSkeleton{
			patch:  patch,
			unsafe: true,
		})
	}
	return c
}

// WithConnectionDetails adds the following connection details to this
// composeTemplateSkeleton.
func (c *composeTemplateSkeleton) WithConnectionDetails(connectionDetails ...xapiextv1.ConnectionDetail) ComposedTemplateSkeleton {
	c.connectionDetails = append(c.connectionDetails, connectionDetails...)
	return c
}

// WithReadinessChecks adds the following readiness checks to this composeTemplateSkeleton.
func (c *composeTemplateSkeleton) WithReadinessChecks(checks ...xapiextv1.ReadinessCheck) ComposedTemplateSkeleton {
	c.readinessChecks = append(c.readinessChecks, checks...)
	return c
}

// ToComposedTemplate converts this composeTemplateSkeleton into a ComposedTemplate.
func (c *composeTemplateSkeleton) ToComposedTemplate() (xapiextv1.ComposedTemplate, error) {
	registeredCompositePaths, err := parseFieldPaths(c.compositionSkeleton.registeredPaths)
	if err != nil {
		return xapiextv1.ComposedTemplate{}, errors.Wrap(err, errParseRegisteredCompositePaths)
	}
	registeredPaths, err := parseFieldPaths(c.registeredPaths)
	if err != nil {
		return xapiextv1.ComposedTemplate{}, errors.Wrap(err, errParseRegisteredComposedPaths)
	}

	c.RegisterAnnotations(KnownResourceAnnotations...)
	c.RegisterLabels(KnownResourceLabels...)

	patches := make([]xapiextv1.Patch, len(c.patches))
	for i, p := range c.patches {
		if !p.unsafe {
			if err := c.validatePatch(&c.patches[i].patch, registeredCompositePaths, registeredPaths); err != nil {
				return xapiextv1.ComposedTemplate{}, errors.Wrapf(err, errFmtInvalidPatch, i)
			}
		}
		patches[i] = p.patch
	}

	base := c.base.Object
	base.SetGroupVersionKind(c.base.GroupVersionKind)

	return xapiextv1.ComposedTemplate{
		Name: c.name,
		Base: runtime.RawExtension{
			Object: base,
		},
		Patches:           patches,
		ConnectionDetails: c.connectionDetails,
		ReadinessChecks:   c.readinessChecks,
	}, nil
}

func (c *composeTemplateSkeleton) validatePatch(patch *xapiextv1.Patch, registeredCompositePaths, registeredPaths []fieldpath.Segments) error {
	patchType := patch.Type
	switch patchType {
	case "", xapiextv1.PatchTypeFromCompositeFieldPath:
		patch.Type = xapiextv1.PatchTypeFromCompositeFieldPath
		return validatePatch(patch, c.compositionSkeleton.composite.Object, c.base.Object, registeredCompositePaths, registeredPaths)
	case xapiextv1.PatchTypeToCompositeFieldPath:
		return validatePatch(patch, c.base.Object, c.compositionSkeleton.composite.Object, registeredPaths, registeredCompositePaths)
	case xapiextv1.PatchTypeCombineFromComposite:
		return validatePatchCombine(patch, c.compositionSkeleton.composite.Object, c.base.Object, registeredCompositePaths, registeredPaths)
	case xapiextv1.PatchTypeCombineToComposite:
		return validatePatchCombine(patch, c.base.Object, c.compositionSkeleton.composite.Object, registeredPaths, registeredCompositePaths)
	case xapiextv1.PatchTypePatchSet:
		return errors.New("patch types not supported")
		/*case xapiextv1.PatchTypeFromEnvironmentFieldPath:
			return errors.New("patch types not supported")
		case xapiextv1.PatchTypeToEnvironmentFieldPath:
			return errors.New("patch types not supported")
		case xapiextv1.PatchTypeCombineFromEnvironment:
			return errors.New("patch types not supported")
		case xapiextv1.PatchTypeCombineToEnvironment:
			return errors.New("patch types not supported")*/
	}
	return errors.Errorf(errUnknownPatchType, patchType)
}
