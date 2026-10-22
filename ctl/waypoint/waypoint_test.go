/*
 * Copyright The Kmesh Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package waypoint

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
	gateway "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapiclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	"kmesh.net/kmesh/pkg/kube"
)

// fakeCLIClient stubs kube.CLIClient with a fake Gateway API clientset.
type fakeCLIClient struct {
	kube.CLIClient
	gatewayClient *gatewayfake.Clientset
}

func (f *fakeCLIClient) GatewayAPI() gatewayapiclient.Interface {
	return f.gatewayClient
}

// The timeout error must name the waypoint the user applied, even when the
// last status poll failed: the polled gateway is empty in that case.
func TestWaitForProgrammedTimeoutNamesWaypoint(t *testing.T) {
	gwClientset := gatewayfake.NewSimpleClientset()
	gwClientset.Fake.PrependReactor("get", "gateways", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("connection refused")
	})
	cli := &fakeCLIClient{gatewayClient: gwClientset}
	gw := &gateway.Gateway{ObjectMeta: metav1.ObjectMeta{Name: "waypoint", Namespace: "default"}}

	err := waitForProgrammed(context.Background(), cli, gw, 0)
	if err == nil {
		t.Fatal("expected a timeout error when every poll fails, got nil")
	}
	if !strings.Contains(err.Error(), "default/waypoint") {
		t.Errorf("timeout error must name the applied waypoint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("timeout error must carry the poll failure, got: %v", err)
	}
}

func TestWaitForProgrammedReady(t *testing.T) {
	programmed := &gateway.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "waypoint", Namespace: "default"},
		Status: gateway.GatewayStatus{
			Conditions: []metav1.Condition{{
				Type:   string(gateway.GatewayConditionProgrammed),
				Status: metav1.ConditionTrue,
			}},
		},
	}
	gwClientset := gatewayfake.NewSimpleClientset()
	gwClientset.Fake.PrependReactor("get", "gateways", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, programmed, nil
	})
	cli := &fakeCLIClient{gatewayClient: gwClientset}
	gw := &gateway.Gateway{ObjectMeta: metav1.ObjectMeta{Name: "waypoint", Namespace: "default"}}

	if err := waitForProgrammed(context.Background(), cli, gw, 5*time.Second); err != nil {
		t.Fatalf("expected no error once the waypoint is Programmed, got: %v", err)
	}
}
