// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eval

import (
	"context"

	"github.com/opentofu/opentofu/internal/addrs"
	"github.com/opentofu/opentofu/internal/lang/eval/internal/configgraph"
	"github.com/opentofu/opentofu/internal/lang/eval/internal/evalglue"
	"github.com/opentofu/opentofu/internal/lang/grapheval"
	"github.com/opentofu/opentofu/internal/providers"
	"github.com/opentofu/opentofu/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// A PlanningOracle provides information from the configuration that is needed
// by the planning engine to help orchestrate the planning process.
type PlanningOracle struct {
	root      evalglue.CompiledModuleInstance
	providers *managedProviders
}

// ProviderInstanceConfig returns a value representing the configuration to
// use when configuring the provider instance with the given address.
//
// The result might contain unknown values, but those should still typically
// be sent to the provider so that it can decide how to deal with them. Some
// providers just immediately fail in that case, but others are able to work
// in a partially-configured mode where some resource types are plannable while
// others need to be deferred to a later plan/apply round.
//
// If the requested provider instance does not exist in the configuration at
// all then this will return nil. That should not occur for any
// provider instance address reported by this package as part of the same
// planning phase, but could occur in subsequent work done by the planning
// phase to deal with resource instances that are in prior state but no longer
// in desired state, if their provider instances have also been removed from
// the desired state at the same time. In that case the planning phase must
// report that the "orphaned" resource instance cannot be planned for deletion
// unless its provider instance is re-added to the configuration.
func (o *PlanningOracle) ProviderInstance(ctx context.Context, addr addrs.AbsProviderInstanceCorrect) (providers.Interface, tfdiags.Diagnostics) {
	return o.providers.ProviderInstance(ctx, addr)
}

// RootOutputs produces the root output values and set resource dependencies
func (o *PlanningOracle) RootOutputs(ctx context.Context) (cty.Value, addrs.Set[addrs.AbsResourceInstance]) {
	ctx = grapheval.ContextWithNewWorker(ctx)

	// Ignore diags, these will be handled by checkAll
	val, _ := o.root.ResultValuer(ctx).Value(ctx)

	contributing := configgraph.ContributingResourceInstances(val)
	deps := addrs.MakeSet[addrs.AbsResourceInstance]()
	for dep := range contributing {
		deps.Add(dep.Addr)
	}

	return configgraph.PrepareOutgoingValue(val), deps
}

func (o *PlanningOracle) Close(ctx context.Context) tfdiags.Diagnostics {
	return o.providers.Close(ctx)
}
