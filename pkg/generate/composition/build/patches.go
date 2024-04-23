package build

import (
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	xapiextv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/mproffitt/crossbuilder/pkg/generate/utils"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

type patchSkeleton struct {
	patch  xapiextv1.Patch
	unsafe bool
}

func validatePatch(patch *xapiextv1.Patch, from, to runtime.Object, fromKnownPaths, toKnownPaths []fieldpath.Segments) error {
	if err := ValidateFieldPath(from, utils.StringValue(patch.FromFieldPath), fromKnownPaths); err != nil {
		return errors.Wrap(err, errPatchFromFieldPath)
	}
	if err := ValidateFieldPath(to, utils.StringValue(patch.ToFieldPath), toKnownPaths); err != nil {
		return errors.Wrap(err, errPatchToFieldPath)
	}
	return nil
}

func validatePatchCombine(patch *xapiextv1.Patch, from, to runtime.Object, fromKnownPaths, toKnownPaths []fieldpath.Segments) error {
	if patch.Combine == nil {
		return errors.Errorf(errPatchRequireField, "combine")
	}
	if patch.Combine.Variables == nil {
		return errors.Errorf(errPatchRequireField, "combine.variables")
	}
	if len(patch.Combine.Variables) == 0 {
		return errors.New(errPatchCombineEmptyVariables)
	}

	for i, v := range patch.Combine.Variables {
		if err := ValidateFieldPath(from, v.FromFieldPath, fromKnownPaths); err != nil {
			return errors.Wrapf(err, errFmtPatchCombineVariableFromFieldPath, i)
		}
	}
	return errors.Wrap(ValidateFieldPath(to, utils.StringValue(patch.ToFieldPath), toKnownPaths), errPatchToFieldPath)
}
