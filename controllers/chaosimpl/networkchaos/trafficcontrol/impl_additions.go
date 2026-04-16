// Copyright 2024 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package trafficcontrol

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/networkchaos/podnetworkchaosmanager"
)

// applyCrossRegionLatency injects per-region WAN latency toward the CIDRs listed in
// each RegionLatencyProfile.  One RawIPSet (type hash:net) and one RawTrafficControl
// are appended for every profile so that different regions can experience different
// latencies simultaneously without interfering with each other.
func (impl *Impl) applyCrossRegionLatency(
	_ context.Context,
	m *podnetworkchaosmanager.PodNetworkManager,
	networkchaos *v1alpha1.NetworkChaos,
	ipSetPostFix string,
	device string,
) error {
	spec := networkchaos.Spec.CrossRegionLatency
	if spec == nil {
		return errors.New("crossRegionLatency spec is required for cross-region-latency action")
	}
	if len(spec.Profiles) == 0 {
		return errors.New("crossRegionLatency.profiles must contain at least one entry")
	}

	for i, profile := range spec.Profiles {
		if len(profile.CIDRs) == 0 {
			return fmt.Errorf("profile[%d] (%s): cidrs must be non-empty", i, profile.RegionName)
		}

		// Build a unique IPSet name: "crl<index><postfix>".
		// ipSetPostFix is "tgt" (3 chars) or "src" (3 chars); the prefix is 3 chars;
		// a two-digit index fits comfortably within the 31-char ipset name limit.
		ipSetName := fmt.Sprintf("crl%d%s", i, ipSetPostFix)

		ipSet := v1alpha1.RawIPSet{
			Name:          ipSetName,
			IPSetType:     v1alpha1.NetIPSet,
			Cidrs:         profile.CIDRs,
			RawRuleSource: v1alpha1.RawRuleSource{Source: m.Source},
		}
		m.T.Append(ipSet)

		// Build TcParameter for this profile.
		tcParam := v1alpha1.TcParameter{
			Delay: &v1alpha1.DelaySpec{
				Latency:     profile.Latency,
				Jitter:      profile.Jitter,
				Correlation: profile.Correlation,
			},
		}
		if profile.Loss != "" {
			tcParam.Loss = &v1alpha1.LossSpec{Loss: profile.Loss}
		}
		if profile.BandwidthRate != "" {
			// Use RateSpec (token-bucket rate limit) so the user only needs to
			// supply a rate string, not full BandwidthSpec limit/buffer values.
			tcParam.Rate = &v1alpha1.RateSpec{Rate: profile.BandwidthRate}
		}

		m.T.Append(v1alpha1.RawTrafficControl{
			Type:        v1alpha1.Netem,
			TcParameter: tcParam,
			Source:      m.Source,
			IPSet:       ipSetName,
			Device:      device,
		})
	}
	return nil
}

// applyRedisClusterFailure injects failures scoped exclusively to Redis protocol
// ports (6379 for client traffic, 16379 for the cluster bus) while leaving all
// other pod-to-pod traffic untouched.
//
// The chaos-daemon uses a hash:net,port ipset to match (CIDR, port) pairs, so
// only packets destined to those ports are shaped – no iptables rules are
// required and no existing connections to other services are affected.
func (impl *Impl) applyRedisClusterFailure(
	_ context.Context,
	m *podnetworkchaosmanager.PodNetworkManager,
	networkchaos *v1alpha1.NetworkChaos,
	ipSetPostFix string,
	device string,
) error {
	spec := networkchaos.Spec.RedisClusterFailure
	if spec == nil {
		return errors.New("redisClusterFailure spec is required for redis-cluster-failure action")
	}

	redisPort := uint16(6379)
	if spec.RedisPort > 0 && spec.RedisPort <= 65535 {
		redisPort = uint16(spec.RedisPort)
	}
	clusterBusPort := uint16(16379)
	if spec.ClusterBusPort > 0 && spec.ClusterBusPort <= 65535 {
		clusterBusPort = uint16(spec.ClusterBusPort)
	}

	// Determine which CIDRs to match.  When ExternalRedisAddresses is set the
	// user wants to target specific external Redis nodes; otherwise we use the
	// wildcard so all outbound Redis traffic on the pod is affected.
	cidrs := []string{"0.0.0.0/0"}
	if len(spec.ExternalRedisAddresses) > 0 {
		cidrs = spec.ExternalRedisAddresses
	}

	var cidrAndPorts []v1alpha1.CidrAndPort
	for _, cidr := range cidrs {
		cidrAndPorts = append(cidrAndPorts,
			v1alpha1.CidrAndPort{Cidr: cidr, Port: redisPort},
			v1alpha1.CidrAndPort{Cidr: cidr, Port: clusterBusPort},
		)
	}

	// "rcf" + postfix fits well within the 31-char ipset name limit.
	ipSetName := fmt.Sprintf("rcf%s", ipSetPostFix)

	ipSet := v1alpha1.RawIPSet{
		Name:          ipSetName,
		IPSetType:     v1alpha1.NetPortIPSet,
		CidrAndPorts:  cidrAndPorts,
		RawRuleSource: v1alpha1.RawRuleSource{Source: m.Source},
	}
	m.T.Append(ipSet)

	var tcType v1alpha1.TcType
	var tcParam v1alpha1.TcParameter

	switch spec.Mode {
	case v1alpha1.RedisPartitionMode:
		// 100% packet loss simulates a complete partition of the Redis nodes.
		tcType = v1alpha1.Netem
		tcParam = v1alpha1.TcParameter{
			Loss: &v1alpha1.LossSpec{Loss: "100"},
		}
	case v1alpha1.RedisLatencyMode:
		if spec.Latency == nil {
			return errors.New("redisClusterFailure.latency is required for latency mode")
		}
		tcType = v1alpha1.Netem
		tcParam = v1alpha1.TcParameter{Delay: spec.Latency}
	case v1alpha1.RedisBandwidthMode:
		if spec.Bandwidth == nil {
			return errors.New("redisClusterFailure.bandwidth is required for bandwidth mode")
		}
		tcType = v1alpha1.Bandwidth
		tcParam = v1alpha1.TcParameter{Bandwidth: spec.Bandwidth}
	default:
		return fmt.Errorf("unknown redis failure mode: %q", spec.Mode)
	}

	m.T.Append(v1alpha1.RawTrafficControl{
		Type:        tcType,
		TcParameter: tcParam,
		Source:      m.Source,
		IPSet:       ipSetName,
		Device:      device,
	})
	return nil
}

// Ensure impl_additions.go is not accidentally imported without impl.go.
var _ *podnetworkchaosmanager.PodNetworkManager // compiler guard
