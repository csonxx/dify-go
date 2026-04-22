package server

import (
	"strings"
	"time"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) syncLinkedRAGPipelineDatasetFromWorkflow(app state.App, workflow state.WorkflowState, user state.User, now time.Time) error {
	if strings.TrimSpace(app.ID) == "" || strings.TrimSpace(app.WorkspaceID) == "" {
		return nil
	}

	dataset, ok := s.store.FindRAGPipelineDataset(app.WorkspaceID, app.ID)
	if !ok {
		return nil
	}

	patch := ragPipelineDatasetPatchFromWorkflow(map[string]any{
		"graph": workflow.Graph,
	})
	if len(patch) == 0 {
		return nil
	}

	_, err := s.store.PatchDataset(dataset.ID, dataset.WorkspaceID, patch, user, now)
	return err
}
