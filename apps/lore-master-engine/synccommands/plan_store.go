package synccommands

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/confluenceplatform"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncexecution"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// keptPlans is how many unexecuted plans the engine remembers; the oldest goes first.
// A person plans, looks, executes; nobody needs the plan from sixteen plans ago.
const keptPlans = 16

// storedPlan is a plan with everything executing it needs.
type storedPlan struct {
	id        string
	sessionID string
	root      string
	output    workspacesettings.Output
	space     platformport.SpaceRef
	plan      syncplanning.SyncPlan
	prepared  syncexecution.Prepared
	// template is the layout every page is put in; empty for the platform's own.
	template confluenceplatform.Template
	// state says where the pages' sync state is recorded (lore-master.json) and what the
	// files carried before the sync, so the write-back migrates it.
	state syncexecution.StateTarget
}

// PlanStore keeps plans between sync/plan and sync/execute.
type PlanStore struct {
	mu    sync.Mutex
	plans map[string]*storedPlan
	order []string
}

// NewPlanStore makes an empty store.
func NewPlanStore() *PlanStore {
	return &PlanStore{plans: map[string]*storedPlan{}}
}

func (s *PlanStore) put(plan *storedPlan) string {
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	plan.id = hex.EncodeToString(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.id] = plan
	s.order = append(s.order, plan.id)
	for len(s.order) > keptPlans {
		delete(s.plans, s.order[0])
		s.order = s.order[1:]
	}

	return plan.id
}

// take removes and returns the plan: a plan is executed once, since executing changes
// the pages and versions it was made from.
func (s *PlanStore) take(id string) (*storedPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	if !ok {
		return nil, rpcprotocol.Errorf(rpcprotocol.CodeUnknownPlan, "no plan %q; it was executed already, replaced by newer plans, or the engine restarted; plan again", id)
	}
	delete(s.plans, id)

	return plan, nil
}
