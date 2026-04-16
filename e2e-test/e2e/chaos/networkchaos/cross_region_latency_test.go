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

// Package networkchaos contains e2e tests for NetworkChaos actions.
package networkchaos

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/e2e-test/e2e/util"
)

// TestCrossRegionLatency verifies that the cross-region-latency action injects
// per-region WAN latency toward specific CIDRs without affecting traffic to
// other destinations.
var _ = Describe("NetworkChaos cross-region-latency", func() {
	var (
		ctx       context.Context
		ns        string
		cli       client.Client
		cancelCtx context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancelCtx = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancelCtx)

		var err error
		cli, err = util.NewClient()
		Expect(err).NotTo(HaveOccurred())

		ns = util.CreateTestNamespace(ctx, cli)
		DeferCleanup(func() { util.DeleteTestNamespace(ctx, cli, ns) })
	})

	Context("with a single region profile", func() {
		It("should inject delay toward the specified CIDR", func() {
			By("deploying a test pod")
			pod := util.NewNetworkTestPod(ns, "nc-crl-src")
			Expect(cli.Create(ctx, pod)).To(Succeed())
			Expect(util.WaitPodRunning(ctx, cli, pod)).To(Succeed())

			By("creating a cross-region-latency NetworkChaos")
			chaos := &v1alpha1.NetworkChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "crl-test",
					Namespace: ns,
				},
				Spec: v1alpha1.NetworkChaosSpec{
					Action: v1alpha1.CrossRegionLatencyAction,
					PodSelector: v1alpha1.PodSelector{
						Selector: v1alpha1.PodSelectorSpec{
							Namespaces: []string{ns},
							LabelSelectors: map[string]string{
								"app": pod.Labels["app"],
							},
						},
						Mode: v1alpha1.OneMode,
					},
					CrossRegionLatency: &v1alpha1.CrossRegionLatencySpec{
						Profiles: []v1alpha1.RegionLatencyProfile{
							{
								RegionName: "us-east-1",
								// RFC 5737 documentation range – safe to use in tests.
								CIDRs:   []string{"203.0.113.0/24"},
								Latency: "100ms",
								Jitter:  "10ms",
							},
						},
					},
				},
			}
			Expect(cli.Create(ctx, chaos)).To(Succeed())
			DeferCleanup(func() { _ = cli.Delete(ctx, chaos) })

			By("waiting for the chaos to be injected")
			Eventually(func(g Gomega) {
				g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(chaos), chaos)).To(Succeed())
				g.Expect(chaos.Status.Conditions).NotTo(BeEmpty())
			}, 30*time.Second, time.Second).Should(Succeed())

			// TODO: measure actual RTT to 203.0.113.1 from within the pod and
			// assert that it is ≥ 90ms (100ms - jitter).  This requires a
			// helper that exec's `ping` inside the pod and parses the output.
			// Tracked in https://github.com/chaos-mesh/chaos-mesh/issues/4892
		})
	})

	Context("with multiple region profiles", func() {
		It("should inject different delays toward each region's CIDRs", func() {
			Skip("TODO: implement multi-profile latency verification – tracked in issue #4892")
		})
	})
})

// TestRedisClusterFailure verifies that the redis-cluster-failure action
// disrupts only Redis protocol ports (6379 + 16379) while leaving other
// traffic on the same pod unaffected.
var _ = Describe("NetworkChaos redis-cluster-failure", func() {
	var (
		ctx       context.Context
		ns        string
		cli       client.Client
		cancelCtx context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancelCtx = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancelCtx)

		var err error
		cli, err = util.NewClient()
		Expect(err).NotTo(HaveOccurred())

		ns = util.CreateTestNamespace(ctx, cli)
		DeferCleanup(func() { util.DeleteTestNamespace(ctx, cli, ns) })
	})

	Context("partition mode", func() {
		It("should drop all packets to Redis ports", func() {
			By("deploying a test pod")
			pod := util.NewNetworkTestPod(ns, "nc-rcf-src")
			Expect(cli.Create(ctx, pod)).To(Succeed())
			Expect(util.WaitPodRunning(ctx, cli, pod)).To(Succeed())

			By("creating a redis-cluster-failure NetworkChaos in partition mode")
			chaos := &v1alpha1.NetworkChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rcf-partition-test",
					Namespace: ns,
				},
				Spec: v1alpha1.NetworkChaosSpec{
					Action: v1alpha1.RedisClusterFailureAction,
					PodSelector: v1alpha1.PodSelector{
						Selector: v1alpha1.PodSelectorSpec{
							Namespaces: []string{ns},
							LabelSelectors: map[string]string{
								"app": pod.Labels["app"],
							},
						},
						Mode: v1alpha1.OneMode,
					},
					RedisClusterFailure: &v1alpha1.RedisClusterFailureSpec{
						Mode: v1alpha1.RedisPartitionMode,
					},
				},
			}
			Expect(cli.Create(ctx, chaos)).To(Succeed())
			DeferCleanup(func() { _ = cli.Delete(ctx, chaos) })

			By("waiting for the chaos to be injected")
			Eventually(func(g Gomega) {
				g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(chaos), chaos)).To(Succeed())
				g.Expect(chaos.Status.Conditions).NotTo(BeEmpty())
			}, 30*time.Second, time.Second).Should(Succeed())

			// TODO: connect to the Redis pod on port 6379 and assert the
			// connection times out, then verify port 80 (if serving HTTP)
			// on the same pod is still reachable.
			// Tracked in https://github.com/chaos-mesh/chaos-mesh/issues/4892
		})
	})

	Context("latency mode", func() {
		It("should add delay to Redis port traffic only", func() {
			Skip("TODO: implement Redis latency mode verification – tracked in issue #4892")
		})
	})

	Context("bandwidth mode", func() {
		It("should throttle Redis port traffic", func() {
			Skip("TODO: implement Redis bandwidth mode verification – tracked in issue #4892")
		})
	})
})
