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

// NOTE: This file contains DeepCopy methods for types introduced by the
// cross-region-latency and redis-cluster-failure features.  They follow the
// same pattern as the methods auto-generated in zz_generated.deepcopy.go and
// should be removed once `make generate` is run and the generated file is
// updated.

package v1alpha1

// DeepCopyInto copies all properties of CrossRegionLatencySpec into another
// CrossRegionLatencySpec instance.
func (in *CrossRegionLatencySpec) DeepCopyInto(out *CrossRegionLatencySpec) {
	*out = *in
	if in.Profiles != nil {
		in, out := &in.Profiles, &out.Profiles
		*out = make([]RegionLatencyProfile, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of CrossRegionLatencySpec.
func (in *CrossRegionLatencySpec) DeepCopy() *CrossRegionLatencySpec {
	if in == nil {
		return nil
	}
	out := new(CrossRegionLatencySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of RegionLatencyProfile into another
// RegionLatencyProfile instance.
func (in *RegionLatencyProfile) DeepCopyInto(out *RegionLatencyProfile) {
	*out = *in
	if in.CIDRs != nil {
		in, out := &in.CIDRs, &out.CIDRs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy creates a deep copy of RegionLatencyProfile.
func (in *RegionLatencyProfile) DeepCopy() *RegionLatencyProfile {
	if in == nil {
		return nil
	}
	out := new(RegionLatencyProfile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of RedisClusterFailureSpec into another
// RedisClusterFailureSpec instance.
func (in *RedisClusterFailureSpec) DeepCopyInto(out *RedisClusterFailureSpec) {
	*out = *in
	if in.Latency != nil {
		in, out := &in.Latency, &out.Latency
		*out = new(DelaySpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Bandwidth != nil {
		in, out := &in.Bandwidth, &out.Bandwidth
		*out = new(BandwidthSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ExternalRedisAddresses != nil {
		in, out := &in.ExternalRedisAddresses, &out.ExternalRedisAddresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy creates a deep copy of RedisClusterFailureSpec.
func (in *RedisClusterFailureSpec) DeepCopy() *RedisClusterFailureSpec {
	if in == nil {
		return nil
	}
	out := new(RedisClusterFailureSpec)
	in.DeepCopyInto(out)
	return out
}
