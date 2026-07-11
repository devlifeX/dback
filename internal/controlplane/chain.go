package controlplane

import (
	"fmt"

	"dback/internal/operation"
)

type ChainContext struct {
	recordIDs []string
}

func (c ChainContext) WithResult(res operation.Result) ChainContext {
	out := c
	for _, a := range res.Artifacts {
		if a.Type == operation.ArtifactExportRecord && a.ID != "" {
			out.recordIDs = append(out.recordIDs, a.ID)
		}
	}
	return out
}

func (c ChainContext) ResolveUploadParams(spec operation.Spec) (operation.Spec, error) {
	if spec.Kind != operation.KindUpload {
		return spec, nil
	}
	params, ok := spec.Params.(operation.UploadParams)
	if !ok {
		return spec, fmt.Errorf("upload params type mismatch")
	}
	if len(params.RecordIDs) == 0 && len(c.recordIDs) > 0 {
		params.RecordIDs = append([]string(nil), c.recordIDs...)
		spec.Params = params
	}
	return spec, nil
}
