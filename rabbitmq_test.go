// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import "testing"

// Assert the concrete transport satisfies the Transport interface.
var _ Transport = (*rabbitMQTransport)(nil)

func TestInvokeRoutingKey(t *testing.T) {
	got := invokeRoutingKey("user-1", "llm.chat")
	want := "invoke.griffino.router.user-1.llm.chat.v1"
	if got != want {
		t.Fatalf("invokeRoutingKey = %q, want %q", got, want)
	}
}

func TestDispatchRoutingKey(t *testing.T) {
	got := dispatchRoutingKey("user-1", "task.created")
	want := "dispatch.griffino.router.user-1.task.created.v1"
	if got != want {
		t.Fatalf("dispatchRoutingKey = %q, want %q", got, want)
	}
}

func TestNodeResultRoutingKey(t *testing.T) {
	got := nodeResultRoutingKey("user-1")
	want := "dispatch.griffino.router.user-1.node.result.v1"
	if got != want {
		t.Fatalf("nodeResultRoutingKey = %q, want %q", got, want)
	}
}

func TestProviderQueueName(t *testing.T) {
	got := providerQueueName("my.plugin", "cap-1")
	want := "plugin.my.plugin.cap-1"
	if got != want {
		t.Fatalf("providerQueueName = %q, want %q", got, want)
	}
}

func TestConsumerQueueNameDotsToUnderscores(t *testing.T) {
	got := consumerQueueName("my.plugin", "task.node.created")
	want := "plugin.my.plugin.consumer.task_node_created"
	if got != want {
		t.Fatalf("consumerQueueName = %q, want %q", got, want)
	}
}

func TestActionQueueName(t *testing.T) {
	got := actionQueueName("my.plugin")
	want := "action.my.plugin"
	if got != want {
		t.Fatalf("actionQueueName = %q, want %q", got, want)
	}
}

func TestActionIDFromRoutingKey(t *testing.T) {
	cases := []struct {
		name       string
		routingKey string
		want       string
	}{
		{"simple", "action.plugin.user.do_thing.v1", "do_thing"},
		{"plugin id with dots", "action.my.plugin.user-1.restart.v1", "restart"},
		{"empty", "", ""},
		{"too few segments", "action.plugin.v1", ""},
		{"exactly five", "a.b.c.d.e", "d"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := actionIDFromRoutingKey(tc.routingKey); got != tc.want {
				t.Fatalf("actionIDFromRoutingKey(%q) = %q, want %q", tc.routingKey, got, tc.want)
			}
		})
	}
}
