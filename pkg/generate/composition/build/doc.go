package build

// This force imports a bunch of packages that will be used in plugins

import (
	_ "github.com/giantswarm/crossplane-fn-network-discovery/pkg/input/v1beta1"
	_ "github.com/upbound/provider-aws/apis"
	_ "github.com/upbound/provider-azure/apis"
	_ "github.com/upbound/provider-gcp/apis"
)
