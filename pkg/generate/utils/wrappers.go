package utils

import xpt "github.com/crossplane-contrib/function-patch-and-transform/input/v1beta1"

func ToPatchPolicy(policy xpt.ToFieldPathPolicy) *xpt.PatchPolicy {
	return &xpt.PatchPolicy{
		ToFieldPath: &policy,
	}
}

func FromPatch(from, to string) xpt.ComposedPatch {
	return xpt.ComposedPatch{
		Type: xpt.PatchTypeFromCompositeFieldPath,
		Patch: xpt.Patch{
			FromFieldPath: &from,
			ToFieldPath:   &to,
		},
	}
}

func ToPatch(to, from string) xpt.ComposedPatch {
	return xpt.ComposedPatch{
		Type: xpt.PatchTypeToCompositeFieldPath,
		Patch: xpt.Patch{
			ToFieldPath:   &to,
			FromFieldPath: &from,
		},
	}
}

func FromPatchMergeObjects(from, to string) xpt.ComposedPatch {
	return xpt.ComposedPatch{
		Type: xpt.PatchTypeFromCompositeFieldPath,
		Patch: xpt.Patch{
			FromFieldPath: &from,
			ToFieldPath:   &to,
			Policy:        ToPatchPolicy(xpt.ToFieldPathPolicyMergeObjects),
		},
	}
}

func ToPatchMergeObjects(to, from string) xpt.ComposedPatch {
	return xpt.ComposedPatch{
		Type: xpt.PatchTypeToCompositeFieldPath,
		Patch: xpt.Patch{
			ToFieldPath:   &to,
			FromFieldPath: &from,
			Policy:        ToPatchPolicy(xpt.ToFieldPathPolicyMergeObjects),
		},
	}
}
