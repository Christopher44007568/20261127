package main

import "testing"

func TestRefCountDefersUntilFrameBoundary(t *testing.T) {
	m := NewManager(PolicyCorrect)
	if err := m.BeginFrame(0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create("tex", 2, 2); err != nil {
		t.Fatal(err)
	}

	producer, err := m.Acquire("tex", "producer")
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := m.Acquire("tex", "consumer")
	if err != nil {
		t.Fatal(err)
	}
	if state, refs, _, ok := m.Info("tex"); !ok || state != ResourceLive || refs != 2 {
		t.Fatalf("unexpected initial state: state=%s refs=%d ok=%t", state, refs, ok)
	}

	if err := m.Release(producer); err != nil {
		t.Fatal(err)
	}
	if state, refs, _, _ := m.Info("tex"); state != ResourceLive || refs != 1 {
		t.Fatalf("resource retired while consumer lease lived: state=%s refs=%d", state, refs)
	}
	if m.QueueLen() != 0 {
		t.Fatalf("queue should be empty while a lease lives, got %d", m.QueueLen())
	}

	if err := m.Release(consumer); err != nil {
		t.Fatal(err)
	}
	if state, refs, _, _ := m.Info("tex"); state != ResourceRetired || refs != 0 {
		t.Fatalf("resource should be retired at frame end: state=%s refs=%d", state, refs)
	}
	if m.QueueLen() != 1 {
		t.Fatalf("retired resource missing from deferred queue: %d", m.QueueLen())
	}
	if _, err := m.Acquire("tex", "too-late"); err == nil {
		t.Fatal("retired resource was acquired again")
	}

	if err := m.BeginFrame(1); err != nil {
		t.Fatal(err)
	}
	if m.QueueLen() != 0 {
		t.Fatalf("frame boundary did not drain queue: %d", m.QueueLen())
	}
	if state, _, _, ok := m.Info("tex"); ok || state != ResourceReclaimed {
		t.Fatalf("resource was not reclaimed at boundary: state=%s ok=%t", state, ok)
	}
}

func TestEarlyPolicyReclaimsInsideFrame(t *testing.T) {
	m := NewManager(PolicyEarly)
	if err := m.BeginFrame(0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create("tex", 2, 2); err != nil {
		t.Fatal(err)
	}
	lease, err := m.Acquire("tex", "only-pass")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Release(lease); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := m.Info("tex"); ok {
		t.Fatal("immediate policy kept the resource alive")
	}
	if _, err := m.Acquire("tex", "later-pass"); err == nil {
		t.Fatal("later pass acquired an immediately reclaimed resource")
	}
	if m.Stats.FailedAcquires != 1 {
		t.Fatalf("failed acquire not counted: %d", m.Stats.FailedAcquires)
	}
}

func TestExtraDelayRetainsOnePastFrame(t *testing.T) {
	m := NewManager(PolicyLate)
	if err := m.BeginFrame(0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create("tex", 2, 2); err != nil {
		t.Fatal(err)
	}
	lease, err := m.Acquire("tex", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Release(lease); err != nil {
		t.Fatal(err)
	}
	if err := m.BeginFrame(1); err != nil {
		t.Fatal(err)
	}
	if state, _, _, ok := m.Info("tex"); !ok || state != ResourceRetired {
		t.Fatalf("extra-delay policy reclaimed too soon: state=%s ok=%t", state, ok)
	}
	if m.QueueLen() != 1 {
		t.Fatalf("late resource should remain queued: %d", m.QueueLen())
	}
	if err := m.BeginFrame(2); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := m.Info("tex"); ok {
		t.Fatal("extra-delay resource was not reclaimed at the later boundary")
	}
	if m.QueueLen() != 0 {
		t.Fatalf("queue did not drain at later boundary: %d", m.QueueLen())
	}
}

func TestReuseRetiredRemovesQueueEntry(t *testing.T) {
	m := NewManager(PolicyLate)
	if err := m.BeginFrame(0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create("tex", 2, 2); err != nil {
		t.Fatal(err)
	}
	lease, _ := m.Acquire("tex", "pass")
	_ = m.Release(lease)
	if _, err := m.ReuseRetired("tex"); err != nil {
		t.Fatal(err)
	}
	if state, refs, _, ok := m.Info("tex"); !ok || state != ResourceLive || refs != 0 {
		t.Fatalf("reused resource is not live: state=%s refs=%d ok=%t", state, refs, ok)
	}
	if m.QueueLen() != 0 {
		t.Fatalf("reuse left a stale queue entry: %d", m.QueueLen())
	}
	if m.Stats.ReusedRetired != 1 {
		t.Fatalf("stale reuse not counted: %d", m.Stats.ReusedRetired)
	}
}
