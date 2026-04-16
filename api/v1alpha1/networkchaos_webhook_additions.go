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

package v1alpha1

import (
	"fmt"
	"net"
	"strconv"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate validates a CrossRegionLatencySpec, checking that every profile has
// non-empty CIDRs, valid CIDR notation, a non-empty latency, and (if set) a
// loss value in the [0, 100] range.
func (in *CrossRegionLatencySpec) Validate(root interface{}, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(in.Profiles) == 0 {
		allErrs = append(allErrs, field.Required(
			path.Child("profiles"),
			"at least one region latency profile is required",
		))
		return allErrs
	}
	for i, p := range in.Profiles {
		pPath := path.Child("profiles").Index(i)
		if len(p.CIDRs) == 0 {
			allErrs = append(allErrs, field.Required(
				pPath.Child("cidrs"),
				fmt.Sprintf("profile[%d] (%s): cidrs must be non-empty", i, p.RegionName),
			))
		}
		for j, cidr := range p.CIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				allErrs = append(allErrs, field.Invalid(
					pPath.Child("cidrs").Index(j), cidr,
					fmt.Sprintf("invalid CIDR: %v", err),
				))
			}
		}
		if p.Latency == "" {
			allErrs = append(allErrs, field.Required(
				pPath.Child("latency"),
				fmt.Sprintf("profile[%d] (%s): latency must be set", i, p.RegionName),
			))
		}
		if p.Loss != "" {
			v, err := strconv.ParseFloat(p.Loss, 64)
			if err != nil || v < 0 || v > 100 {
				allErrs = append(allErrs, field.Invalid(
					pPath.Child("loss"), p.Loss,
					"loss must be a float in [0, 100]",
				))
			}
		}
	}
	return allErrs
}

// Validate validates a RedisClusterFailureSpec, checking that the mode is
// known and that the mode-specific sub-spec is present.
func (in *RedisClusterFailureSpec) Validate(root interface{}, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	switch in.Mode {
	case RedisPartitionMode:
		// no extra fields required
	case RedisLatencyMode:
		if in.Latency == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("latency"),
				"latency spec is required for latency mode",
			))
		}
	case RedisBandwidthMode:
		if in.Bandwidth == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("bandwidth"),
				"bandwidth spec is required for bandwidth mode",
			))
		}
	default:
		allErrs = append(allErrs, field.NotSupported(
			path.Child("mode"), in.Mode,
			[]string{
				string(RedisPartitionMode),
				string(RedisLatencyMode),
				string(RedisBandwidthMode),
			},
		))
	}
	if in.RedisPort > 65535 {
		allErrs = append(allErrs, field.Invalid(
			path.Child("redisPort"), in.RedisPort, "port must be <= 65535",
		))
	}
	if in.ClusterBusPort > 65535 {
		allErrs = append(allErrs, field.Invalid(
			path.Child("clusterBusPort"), in.ClusterBusPort, "port must be <= 65535",
		))
	}
	for i, addr := range in.ExternalRedisAddresses {
		_, _, cidrErr := net.ParseCIDR(addr)
		ipOK := net.ParseIP(addr) != nil
		if cidrErr != nil && !ipOK {
			allErrs = append(allErrs, field.Invalid(
				path.Child("externalRedisAddresses").Index(i), addr,
				"must be a valid CIDR (e.g. 10.0.0.0/8) or IP address",
			))
		}
	}
	return allErrs
}
